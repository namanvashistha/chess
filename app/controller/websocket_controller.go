package controller

import (
	"chess-engine/app/domain/dto"
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

// HandleWebSocket upgrades HTTP connection to WebSocket and manages communication.
func (wsCtrl WebSocketControllerImpl) HandleWebSocket(c *gin.Context) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			// Allow all connections for simplicity; customize as needed.
			return true
		},
	}

	// Upgrade HTTP to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to establish WebSocket connection"})
		return
	}

	// Register the client with the WebSocket service
	wsCtrl.svc.RegisterClient(conn)

	defer wsCtrl.svc.UnregisterClient(conn) // Ensure cleanup on disconnect

	// Listen for messages from the client
	for {
		log.Info("Waiting for WebSocket message...")
		var message dto.WebSocketMessage
		if err := conn.ReadJSON(&message); err != nil {
			// Connection closed or invalid message
			log.Info("Closing WebSocket connection Error...")
			break
		}

		log.Infof("Received message: %+v", message)

		// Process the received message
		wsCtrl.svc.ProcessMove(message)
	}
}

// WebSocketControllerInit initializes the WebSocket controller
func WebSocketControllerInit(wsService service.WebSocketService) *WebSocketControllerImpl {
	return &WebSocketControllerImpl{
		svc: wsService,
	}
}
