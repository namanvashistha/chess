package router

import (
	"chess-engine/config"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocket Upgrader: Upgrades HTTP connections to WebSocket connections.
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for simplicity; consider tightening this for security.
	},
}

// Init initializes the router with routes for API and WebSocket.
func Init(init *config.Initialization) *gin.Engine {

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Serve static files
	router.Static("/static", "./app/static")

	// Route to render chessboard
	router.GET("/game", func(c *gin.Context) {
		c.File("./app/static/html/chess_board.html")
	})

	// WebSocket route
	router.GET("/ws", init.SocketCtrl.HandleWebSocket)

	// API routes
	api := router.Group("/api")
	{
		user := api.Group("/user")
		{
			user.GET("", init.UserCtrl.GetAllUserData)
			user.POST("", init.UserCtrl.AddUserData)
			user.GET("/:userID", init.UserCtrl.GetUserById)
			user.PUT("/:userID", init.UserCtrl.UpdateUserData)
			user.DELETE("/:userID", init.UserCtrl.DeleteUser)
		}
		chess := api.Group("/chess")
		{
			chess.GET("", init.ChessCtrl.GetAllChess)
			chess.GET("/:chessID", init.ChessCtrl.GetChessById)
			chess.GET("/state/:gameId", init.ChessCtrl.GetChessState)
			chess.GET("/state/init", init.ChessCtrl.CreateChessState)
			chess.POST("/state/move", init.ChessCtrl.MakeMove)
		}
	}

	return router
}

// // handleWebSocketConnection upgrades the connection and delegates to the WebSocketController.
// func handleWebSocketConnection(c *gin.Context, websocketController controller.WebSocketController) {
// 	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upgrade connection"})
// 		return
// 	}
// 	defer conn.Close()

// 	websocketController.HandleWebSocket(c)
// }
