package service

import (
	"chess-engine/app/constant"
	"chess-engine/app/domain/dao"
	"chess-engine/app/domain/dto"
	"chess-engine/app/engine"
	"chess-engine/app/pkg"
	"chess-engine/app/repository"
	"fmt"
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
	game, _ = ws.chessRepository.FindChessGameById(gameId)
	// Fetch from cache
	gameState, err := ws.chessRepository.GetChessStateStateFromCache(fmt.Sprint(game.ChessStateId))
	if err != nil || gameState.ID == 0 {
		// Fallback to DB if cache miss
		log.Info("Cache miss. Fetching from database.")
		gameState, err = ws.chessRepository.GetChessStateStateFromDB(fmt.Sprint(game.ChessStateId))
		if err != nil {
			log.Error("Error fetching game state:", err)
			pkg.PanicException(constant.DataNotFound)
		}
		// Save to cache after fetching from DB
		_ = ws.chessRepository.SaveChessStateToCache(&gameState)
	}
	var move dto.Move
	move.Destination = message.Payload.(map[string]interface{})["destination"].(string)
	move.Source = message.Payload.(map[string]interface{})["source"].(string)
	move.Piece = message.Payload.(map[string]interface{})["piece"].(string)
	err = engine.MakeMove(&gameState, move)
	status := "success"
	if err != nil {
		status = "error"
		log.Error("Error processing move:", err)
	} else {
		_ = ws.chessRepository.SaveChessStateToCache(&gameState)
		_ = ws.chessRepository.SaveChessStateToDB(&gameState)
	}

	// Build response
	allowedMoves := engine.GetAllowedMoves(gameState)
	boardlayout := engine.GetBoardLayout()
	gameState.AllowedMoves = allowedMoves
	gameState.BoardLayout = boardlayout
	game.ChessState = gameState
	response := dto.WebSocketMessage{
		Type:    "game_update",
		Status:  status,
		Payload: game,
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
