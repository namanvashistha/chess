package controller

import (
	"chess-engine/app/domain/dto"
	"chess-engine/app/pkg"
	"chess-engine/app/service"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

type WebSocketController interface {
	HandleWebSocket(c *gin.Context)
}

type WebSocketControllerImpl struct {
	svc service.WebSocketService
}

// upgrader is package-level: building it per request allocated a new one for
// every connection for no reason.
var upgrader = websocket.Upgrader{
	CheckOrigin: pkg.CheckWebSocketOrigin,
}

// HandleWebSocket upgrades HTTP connection to WebSocket and manages communication.
func (wsCtrl WebSocketControllerImpl) HandleWebSocket(c *gin.Context) {
	gameID := c.Param("gameId")
	if gameID == "" {
		// The `return` here used to be commented out, so a request with no game
		// id got a 400 written to it and *then* had its connection upgraded.
		c.JSON(http.StatusBadRequest, gin.H{"error": "game_id is required"})
		return
	}

	// Upgrade HTTP to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		// Upgrade already wrote an HTTP error response; adding another would
		// trigger a superfluous-WriteHeader warning.
		log.Error("Failed to establish WebSocket connection: ", err)
		return
	}

	// Register the client with the WebSocket service
	wsCtrl.svc.RegisterClient(gameID, conn)

	defer wsCtrl.svc.UnregisterClient(gameID, conn) // Ensure cleanup on disconnect

	// If the bot has the move (e.g. it drew White), play it now that someone is watching.
	go wsCtrl.svc.MaybePlayBotMove(gameID)

	// Listen for messages from the client
	for {
		var message dto.WebSocketMessage
		if err := conn.ReadJSON(&message); err != nil {
			// Connection closed or invalid message
			log.Info("Closing WebSocket connection: ", err)
			break
		}

		wsCtrl.handleMessage(gameID, message)
	}
}

// handleMessage processes one client message, containing any panic to this
// connection.
//
// This loop runs on its own goroutine, so gin.Recovery() does not cover it: an
// unrecovered panic here (e.g. a bad type assertion on a client-supplied
// payload) terminated the entire process and every other live game with it.
func (wsCtrl WebSocketControllerImpl) handleMessage(gameID string, message dto.WebSocketMessage) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Errorf("recovered panic while processing message for game %s: %v", gameID, rec)
		}
	}()

	wsCtrl.svc.ProcessMove(gameID, message)
}

// WebSocketControllerInit initializes the WebSocket controller
func WebSocketControllerInit(wsService service.WebSocketService) *WebSocketControllerImpl {
	return &WebSocketControllerImpl{
		svc: wsService,
	}
}
