package router

import (
	"chess-engine/config"
	"net/http"
	"strings"

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

	// Legacy assets (piece SVGs, etc.) and the SvelteKit build assets.
	router.Static("/static", "./app/static")
	router.Static("/_app", "./web/build/_app")

	// SPA shell. The SvelteKit client router owns "/", "/game/:id", etc.
	const spaIndex = "./web/build/index.html"
	router.GET("/", func(c *gin.Context) {
		c.File(spaIndex)
	})

	// Legacy bitboard debug page.
	router.GET("/bitboard", func(c *gin.Context) {
		c.File("./app/static/html/bitboard.html")
	})

	// WebSocket route
	router.GET("/ws/:gameId", init.SocketCtrl.HandleWebSocket)

	// API routes
	api := router.Group("/api")
	{
		user := api.Group("/user")
		{
			// user.GET("", init.UserCtrl.GetAllUserData)
			user.POST("", init.UserCtrl.AddUserData)
			user.POST("/me", init.UserCtrl.GetUserByToken)
			user.PUT("/:userID", init.UserCtrl.UpdateUserData)
			user.DELETE("/:userID", init.UserCtrl.DeleteUser)
		}
		chess := api.Group("/chess")
		{
			chess.GET("/game", init.ChessCtrl.GetAllChessGame)
			chess.POST("/game", init.ChessCtrl.CreateChessGame)
			chess.POST("/game/bot", init.ChessCtrl.CreateBotChessGame)
			chess.GET("/game/:gameId", init.ChessCtrl.GetChessGameById)
			chess.POST("/game/join", init.ChessCtrl.JoinChessGame)
			chess.POST("/state/move", init.ChessCtrl.MakeMove)
		}
	}

	// Client-side routes (e.g. /game/123) fall back to the SPA shell; everything
	// under /api and /ws stays a real 404 so the client sees API errors clearly.
	router.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if c.Request.Method == http.MethodGet &&
			!strings.HasPrefix(p, "/api") && !strings.HasPrefix(p, "/ws") &&
			!strings.HasPrefix(p, "/static") && !strings.HasPrefix(p, "/_app") {
			c.File(spaIndex)
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	})

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
