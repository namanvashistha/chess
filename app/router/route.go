package router

import (
	"chess-engine/app/middleware"
	"chess-engine/config"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

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

	requireAuth := middleware.RequireAuth(init.UserRepo)

	// API routes
	api := router.Group("/api")
	{
		user := api.Group("/user")
		{
			user.POST("", init.UserCtrl.AddUserData)
			user.POST("/me", init.UserCtrl.GetUserByToken)
			// Mutating another account is only possible for its owner: these two
			// routes previously had no authentication at all.
			user.PUT("/:userID", requireAuth, init.UserCtrl.UpdateUserData)
			user.DELETE("/:userID", requireAuth, init.UserCtrl.DeleteUser)
		}
		chess := api.Group("/chess")
		{
			chess.GET("/game", init.ChessCtrl.GetAllChessGame)
			chess.POST("/game", init.ChessCtrl.CreateChessGame)
			chess.POST("/game/bot", init.ChessCtrl.CreateBotChessGame)
			chess.POST("/game/local", init.ChessCtrl.CreateLocalChessGame)
			chess.GET("/game/:gameId", init.ChessCtrl.GetChessGameById)
			chess.POST("/game/join", init.ChessCtrl.JoinChessGame)
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
