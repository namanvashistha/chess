package service

import (
	"chess-engine/app/constant"
	"chess-engine/app/domain/dao"
	"chess-engine/app/domain/dto"
	"chess-engine/app/engine"
	"chess-engine/app/pkg"
	"chess-engine/app/repository"
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

// Internal run loop
func (ws *WebSocketServiceImpl) ProcessMove(gameId string, message dto.WebSocketMessage) {
	log.Info("Processing move via WebSocket", message.Payload)

	var game dao.ChessGame
	var err error
	status := "success"
	status_message := ""
	game, err = ws.chessRepository.GetChessGameFromCache(gameId)
	if err != nil || game.ID == 0 {
		// Fallback to DB if cache miss
		log.Info("Cache miss. Fetching from database.")
		game, err = ws.chessRepository.FindChessGameById(gameId)
		if err != nil {
			log.Error("Error fetching game state:", err)
			status = "error"
		}
		// Save to cache after fetching from DB
		_ = ws.chessRepository.SaveChessGameToCache(&game)
	} else {
		log.Info("Fetched game state from cache:", game.ID)
	}
	var move dto.Move
	if err = pkg.BindPayloadToStruct(message.Payload.(map[string]interface{}), &move); err != nil {
		log.Errorf("Failed to unmarshal move: %v", err)
		status = "error"
	}
	user, err := ws.chessRepository.FindUserByToken(move.Token)
	if err != nil {
		status = "error"
		log.Error("Error fetching user by token:", err)
		status_message = err.Error()
		status = "error"
	}
	legalMoves := make(map[uint64]uint64)
	gameStatus := ""
	if lastMove, err := engine.ProcessMove(&game, move, user); err != nil {
		status = "error"
		status_message = err.Error()
		log.Error("Error processing move:", err)
	} else {
		legalMoves, gameStatus = engine.GenerateLegalMovesForAllPositions(game.State)
		gameMove := dao.GameMove{
			GameID: game.ID,
			Move:   lastMove,
		}
		_ = ws.chessRepository.SaveGameMoveToDB(&gameMove)
		game.Moves = append(game.Moves, gameMove)
		if gameStatus == "white_checkmate" {
			game.Winner = "b"
		}

		if gameStatus == "black_checkmate" {
			game.Winner = "w"
		}

		_ = ws.chessRepository.SaveChessGameToCache(&game)
		_ = ws.chessRepository.SaveChessGameToDB(&game)
		_ = ws.chessRepository.SaveGameStateToDB(&game.State)
	}

	game.BoardLayout = engine.GetBoardLayout()
	game.CurrentState = engine.ConvertGameStateToMap(game.State)

	if len(legalMoves) == 0 {
		legalMoves, gameStatus = engine.GenerateLegalMovesForAllPositions(game.State)
	}
	game.LegalMoves = engine.ConvertLegalMovesToMap(legalMoves)
	response := dto.WebSocketMessage{
		Type:    "game_update",
		Status:  status,
		Message: status_message + gameStatus,
		Payload: game,
	}
	ws.BroadcastMessage(gameId, response)

	// If it's now the bot's turn, let it reply through this same pipeline.
	if status == "success" {
		go ws.MaybePlayBotMove(gameId)
	}
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

	move := engine.ChooseGreedyMove(&game)
	if move == nil {
		return
	}

	// A short pause so the reply doesn't feel instant.
	time.Sleep(600 * time.Millisecond)

	ws.ProcessMove(gameId, dto.WebSocketMessage{
		Type: "game_update",
		Payload: map[string]interface{}{
			"piece":       move.Piece,
			"source":      move.Source,
			"destination": move.Destination,
			"game_id":     gameId,
			"token":       constant.BotToken,
		},
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
