package service

import (
	"chess-engine/app/constant"
	"chess-engine/app/domain/dao"
	"chess-engine/app/domain/dto"
	"chess-engine/app/engine"
	"chess-engine/app/pkg"
	"chess-engine/app/repository"
	"fmt"
	"hash/fnv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

type WebSocketService interface {
	RegisterClient(gameId string, conn *websocket.Conn)
	UnregisterClient(gameId string, conn *websocket.Conn)
	BroadcastMessage(gameID string, message dto.WebSocketMessage)
	ProcessMove(gameId string, message dto.WebSocketMessage)
	MaybePlayBotMove(gameId string)
}

type WebSocketServiceImpl struct {
	gameClients     map[string]map[*websocket.Conn]bool // Map[gameID] -> Map[Conn] -> bool
	broadcast       chan gameBroadcastMessage           // Messages tied to a game_id
	register        chan clientRegistration             // Registration with game_id
	unregister      chan clientRegistration             // Unregistration with game_id
	chessRepository repository.ChessRepository
	mutex           sync.Mutex
	gameLocks       [gameLockStripes]sync.Mutex // serializes move application per game
}

// gameLockStripes is the size of the fixed lock table below. It must be a power
// of two so the modulo is a mask.
const gameLockStripes = 256

// lockFor returns the mutex guarding a game so its load→apply→save→broadcast
// runs atomically. Without this, a human move and the bot's reply (or a
// duplicate bot trigger on connect) can load the same cached state and
// overwrite each other, losing moves and corrupting the position.
//
// This is a fixed table of stripes rather than a sync.Map keyed by game id: that
// map was never pruned, so it grew a permanent mutex for every game id ever
// seen -- an unbounded leak keyed on user input. Striping bounds it at a
// constant. Two different games can hash to the same stripe, which only means
// their (already sub-millisecond) move application is briefly serialized.
func (ws *WebSocketServiceImpl) lockFor(gameID string) *sync.Mutex {
	h := fnv.New32a()
	_, _ = h.Write([]byte(gameID))
	return &ws.gameLocks[h.Sum32()%gameLockStripes]
}

type clientRegistration struct {
	GameID string
	Conn   *websocket.Conn
}

type gameBroadcastMessage struct {
	GameID  string
	Message dto.WebSocketMessage
}

// Constructor
func NewWebSocketService(chessRepository repository.ChessRepository) *WebSocketServiceImpl {
	service := &WebSocketServiceImpl{
		gameClients:     make(map[string]map[*websocket.Conn]bool),
		broadcast:       make(chan gameBroadcastMessage),
		register:        make(chan clientRegistration),
		unregister:      make(chan clientRegistration),
		chessRepository: chessRepository,
	}
	go service.run()
	return service
}

// Register a client to a specific game
func (ws *WebSocketServiceImpl) RegisterClient(gameID string, conn *websocket.Conn) {
	ws.register <- clientRegistration{GameID: gameID, Conn: conn}
}

// Unregister a client from a specific game
func (ws *WebSocketServiceImpl) UnregisterClient(gameID string, conn *websocket.Conn) {
	ws.unregister <- clientRegistration{GameID: gameID, Conn: conn}
}

// Broadcast a message to all clients in a specific game
func (ws *WebSocketServiceImpl) BroadcastMessage(gameID string, message dto.WebSocketMessage) {
	ws.broadcast <- gameBroadcastMessage{GameID: gameID, Message: message}
}

// ProcessMove authenticates a client message and applies the move it carries.
func (ws *WebSocketServiceImpl) ProcessMove(gameId string, message dto.WebSocketMessage) {
	// The payload is attacker-controlled. Asserting the type unconditionally
	// (message.Payload.(map[string]interface{})) panicked on any non-object
	// payload, and this runs on the connection's own goroutine, so that panic
	// took down the whole server.
	payload, ok := message.Payload.(map[string]interface{})
	if !ok {
		log.Errorf("Rejecting move for game %s: payload is %T, want object", gameId, message.Payload)
		ws.sendError(gameId, "invalid move payload")
		return
	}

	var move dto.Move
	if err := pkg.BindPayloadToStruct(payload, &move); err != nil {
		log.Errorf("Failed to unmarshal move: %v", err)
		ws.sendError(gameId, "invalid move payload")
		return
	}

	user, err := ws.chessRepository.FindUserByToken(move.Token)
	if err != nil || user.ID == 0 {
		// The token doesn't map to a user (e.g. a stale token after a DB reset).
		// Bail out with a clear message instead of running ProcessMove with a
		// zero user, which would report the misleading "user 0 is not in the game".
		log.Error("Error fetching user by token:", err)
		ws.sendError(gameId, "session expired, please reload")
		return
	}

	ws.applyMove(gameId, move, user)
}

// applyMove runs the load -> validate -> persist -> broadcast pipeline for an
// already-authenticated user. The bot calls this directly rather than round
// -tripping a shared secret through the message payload.
func (ws *WebSocketServiceImpl) applyMove(gameId string, move dto.Move, user dao.User) {
	// Serialize all move application for this game (human + bot) so concurrent
	// calls can't load the same state and clobber each other.
	lk := ws.lockFor(gameId)
	lk.Lock()
	defer lk.Unlock()

	game, err := ws.loadGame(gameId)
	if err != nil {
		// Previously this only set status = "error" and fell through, running the
		// move against a zero-valued game.
		log.Error("Error fetching game state:", err)
		ws.sendError(gameId, "could not load game")
		return
	}

	status := "success"
	statusMessage := ""
	legalMoves := make(map[uint64]uint64)
	gameStatus := ""

	if lastMove, err := engine.ProcessMove(&game, move, user); err != nil {
		status = "error"
		statusMessage = err.Error()
		log.Error("Error processing move:", err)
	} else {
		legalMoves, gameStatus = engine.GenerateLegalMovesForAllPositions(game.State)
		gameMove := dao.GameMove{
			GameID: game.ID,
			Move:   lastMove,
		}
		if gameStatus == "white_checkmate" {
			game.Winner = "b"
		}
		if gameStatus == "black_checkmate" {
			game.Winner = "w"
		}

		// Every one of these writes used to be `_ =`. A database or Redis outage
		// looked exactly like a successful move: the client saw the new position
		// broadcast and only found out on reload that it was never saved.
		if err := ws.persist(&game, &gameMove); err != nil {
			status = "error"
			statusMessage = "move could not be saved, please reload"
			log.Error("Error persisting move:", err)
		} else {
			game.Moves = append(game.Moves, gameMove)
		}
	}

	game.BoardLayout = engine.GetBoardLayout()
	game.CurrentState = engine.ConvertGameStateToMap(game.State)

	// Only regenerate when the move was rejected; `len(legalMoves) == 0` was a
	// bad sentinel because a real checkmate legitimately has no legal moves and
	// so paid for the (expensive) generation twice on every mating move.
	if status != "success" {
		legalMoves, gameStatus = engine.GenerateLegalMovesForAllPositions(game.State)
	}
	legalMoves = engine.FilterMovesByTurn(legalMoves, game.State)
	game.LegalMoves = engine.ConvertLegalMovesToMap(legalMoves)
	response := dto.WebSocketMessage{
		Type:    "game_update",
		Status:  status,
		Message: statusMessage + gameStatus,
		Payload: game,
	}
	ws.BroadcastMessage(gameId, response)

	// If it's now the bot's turn, let it reply through this same pipeline.
	if status == "success" {
		go ws.MaybePlayBotMove(gameId)
	}
}

// loadGame reads a game from the cache, falling back to the database.
func (ws *WebSocketServiceImpl) loadGame(gameId string) (dao.ChessGame, error) {
	game, err := ws.chessRepository.GetChessGameFromCache(gameId)
	if err == nil && game.ID != 0 {
		log.Info("Fetched game state from cache:", game.ID)
		return game, nil
	}

	log.Info("Cache miss. Fetching from database.")
	game, err = ws.chessRepository.FindChessGameById(gameId)
	if err != nil {
		return dao.ChessGame{}, err
	}
	if err := ws.chessRepository.SaveChessGameToCache(&game); err != nil {
		// Non-fatal: the DB is the source of truth, we just lose the cache hit.
		log.Warn("Could not warm game cache: ", err)
	}
	return game, nil
}

// persist writes the move and the resulting game state. The database writes are
// the ones that matter; a cache write failure is logged but not fatal.
func (ws *WebSocketServiceImpl) persist(game *dao.ChessGame, gameMove *dao.GameMove) error {
	if err := ws.chessRepository.SaveGameMoveToDB(gameMove); err != nil {
		return fmt.Errorf("save game move: %w", err)
	}
	if err := ws.chessRepository.SaveGameStateToDB(&game.State); err != nil {
		return fmt.Errorf("save game state: %w", err)
	}
	if err := ws.chessRepository.SaveChessGameToDB(game); err != nil {
		return fmt.Errorf("save game: %w", err)
	}
	if err := ws.chessRepository.SaveChessGameToCache(game); err != nil {
		log.Warn("Could not update game cache: ", err)
	}
	return nil
}

// MaybePlayBotMove checks whether the side to move is the built-in bot and, if
// so, computes a move and replays it through ProcessMove (so it persists and
// broadcasts exactly like a human move). It's a no-op for human turns, finished
// games, and games without a bot, which makes it safe to call after every move
// and on connect. No recursion risk: a bot game has only one bot seat, so after
// the bot moves it's the human's turn and this returns immediately.
func (ws *WebSocketServiceImpl) MaybePlayBotMove(gameId string) {
	game, err := ws.chessRepository.GetChessGameFromCache(gameId)
	if err != nil || game.ID == 0 {
		game, err = ws.chessRepository.FindChessGameById(gameId)
		if err != nil {
			return
		}
	}
	if game.Winner != "" || game.WhiteUser == nil || game.BlackUser == nil {
		return
	}

	botToMove := (game.State.Turn == "w" && game.WhiteUser.Name == constant.BotName) ||
		(game.State.Turn == "b" && game.BlackUser.Name == constant.BotName)
	if !botToMove {
		return
	}

	// Resolve the bot's own user row. The bot used to authenticate by putting
	// constant.BotToken -- a credential hardcoded in the source -- into the move
	// payload and going back through the token lookup. It now applies its move
	// directly as an already-known user, so no shared secret exists to leak.
	botUser, err := ws.chessRepository.FindOrCreateBotUser()
	if err != nil {
		log.Error("Could not resolve bot user for bot move:", err)
		return
	}

	move := engine.ChooseBotMove(&game)
	if move == nil {
		return
	}

	// A short pause so the reply doesn't feel instant.
	time.Sleep(600 * time.Millisecond)

	ws.applyMove(gameId, *move, botUser)
}

// sendError broadcasts an error-status game_update so the client can surface the
// reason without us having to fabricate a game payload.
func (ws *WebSocketServiceImpl) sendError(gameId, message string) {
	ws.BroadcastMessage(gameId, dto.WebSocketMessage{
		Type:    "game_update",
		Status:  "error",
		Message: message,
	})
}

func (ws *WebSocketServiceImpl) run() {
	for {
		select {
		case reg := <-ws.register:
			ws.mutex.Lock()
			if _, exists := ws.gameClients[reg.GameID]; !exists {
				ws.gameClients[reg.GameID] = make(map[*websocket.Conn]bool)
			}
			ws.gameClients[reg.GameID][reg.Conn] = true
			ws.mutex.Unlock()
			log.Infof("Client connected to game %s", reg.GameID)

		case unreg := <-ws.unregister:
			ws.mutex.Lock()
			if clients, exists := ws.gameClients[unreg.GameID]; exists {
				if _, ok := clients[unreg.Conn]; ok {
					delete(clients, unreg.Conn)
					unreg.Conn.Close()
					log.Infof("Client disconnected from game %s", unreg.GameID)

					// Cleanup empty game entries
					if len(clients) == 0 {
						delete(ws.gameClients, unreg.GameID)
						log.Infof("No clients left for game %s. Removed from active games.", unreg.GameID)
					}
				}
			}
			ws.mutex.Unlock()

		case broadcast := <-ws.broadcast:
			ws.mutex.Lock()
			if clients, exists := ws.gameClients[broadcast.GameID]; exists {
				for conn := range clients {
					err := conn.WriteJSON(broadcast.Message)
					if err != nil {
						log.Error("Error broadcasting message to client: ", err)
						conn.Close()
						delete(clients, conn)
					}
				}

				// Cleanup if no clients remain
				if len(clients) == 0 {
					delete(ws.gameClients, broadcast.GameID)
				}
			}
			ws.mutex.Unlock()
		}
	}
}

func WebSocketServiceInit(chessRepository repository.ChessRepository) WebSocketService {
	return NewWebSocketService(chessRepository)
}
