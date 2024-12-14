package service

import (
	"chess-engine/app/constant"
	"chess-engine/app/domain/dao"
	"chess-engine/app/domain/dto"
	"chess-engine/app/engine"
	"chess-engine/app/pkg"
	"chess-engine/app/repository"
	"sync"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

type WebSocketService interface {
	RegisterClient(conn *websocket.Conn)
	UnregisterClient(conn *websocket.Conn)
	BroadcastMessage(message dto.WebSocketMessage)
	ProcessMove(message dto.WebSocketMessage)
}

type WebSocketServiceImpl struct {
	clients         map[*websocket.Conn]bool
	broadcast       chan dto.WebSocketMessage
	register        chan *websocket.Conn
	unregister      chan *websocket.Conn
	chessRepository repository.ChessRepository
	mutex           sync.Mutex
}

func NewWebSocketService(chessRepository repository.ChessRepository) *WebSocketServiceImpl {
	service := &WebSocketServiceImpl{
		clients:         make(map[*websocket.Conn]bool),
		broadcast:       make(chan dto.WebSocketMessage),
		register:        make(chan *websocket.Conn),
		unregister:      make(chan *websocket.Conn),
		chessRepository: chessRepository,
	}
	go service.run()
	return service
}

func (ws *WebSocketServiceImpl) RegisterClient(conn *websocket.Conn) {
	ws.register <- conn
}

func (ws *WebSocketServiceImpl) UnregisterClient(conn *websocket.Conn) {
	ws.unregister <- conn
}

func (ws *WebSocketServiceImpl) BroadcastMessage(message dto.WebSocketMessage) {
	ws.broadcast <- message
}

func (ws *WebSocketServiceImpl) ProcessMove(message dto.WebSocketMessage) {
	log.Info("Processing move via WebSocket", message.Payload)

	var game dao.ChessGame
	var err error
	gameId := message.Payload.(map[string]interface{})["game_id"].(string)
	log.Info("websocket Game ID:", gameId)
	// Fetch from cache
	game, err = ws.chessRepository.GetChessGameStateFromCache(gameId)
	if err != nil || game.ID == 0 {
		// Fallback to DB if cache miss
		log.Info("Cache miss. Fetching from database.")
		game, err = ws.chessRepository.GetChessGameStateFromDB(gameId)
		if err != nil {
			log.Error("Error fetching game state:", err)
			pkg.PanicException(constant.DataNotFound)
		}
		// Save to cache after fetching from DB
		_ = ws.chessRepository.SaveChessGameToCache(&game)
	}

	// Build response
	allowedMoves := engine.GetAllowedMoves(game)
	boardlayout := engine.GetBoardLayout()
	pieceMap := engine.GetPiecesMap()

	response := dto.WebSocketMessage{
		Type: "game_update",
		Payload: map[string]interface{}{
			"board":         game.Board,
			"turn":          game.Turn,
			"status":        game.Status,
			"last_move":     game.LastMove,
			"allowed_moves": allowedMoves,
			"board_layout":  boardlayout,
			"pieces_map":    pieceMap,
		},
	}
	ws.BroadcastMessage(response)
}

func (ws *WebSocketServiceImpl) run() {
	for {
		select {
		case conn := <-ws.register:
			ws.mutex.Lock()
			ws.clients[conn] = true
			ws.mutex.Unlock()
			log.Info("Client connected")

		case conn := <-ws.unregister:
			ws.mutex.Lock()
			if _, ok := ws.clients[conn]; ok {
				delete(ws.clients, conn)
				conn.Close()
				log.Info("Client disconnected")
			}
			ws.mutex.Unlock()

		case message := <-ws.broadcast:
			ws.mutex.Lock()
			for conn := range ws.clients {
				err := conn.WriteJSON(message)
				if err != nil {
					log.Error("Error broadcasting message: ", err)
					conn.Close()
					delete(ws.clients, conn)
				}
			}
			ws.mutex.Unlock()
		}
	}
}

func WebSocketServiceInit(chessRepository repository.ChessRepository) WebSocketService {
	return NewWebSocketService(chessRepository)
}
