package router

import (
	"chess-engine/config"

	"github.com/gin-gonic/gin"
)

func Init(init *config.Initialization) *gin.Engine {

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Serve static files
	router.Static("/static", "./app/static")

	// Route to render chessboard
	router.GET("/chessboard", func(c *gin.Context) {
		c.File("./app/static/html/chess_board.html")
	})

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
			chess.GET("/state", init.ChessCtrl.GetChessState)
			chess.GET("/state/init", init.ChessCtrl.CreateChessState)
			chess.POST("/state/move", init.ChessCtrl.MakeMove)
		}
	}

	return router
}
