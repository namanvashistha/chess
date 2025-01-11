package controller

import (
	"chess-engine/app/service"

	"github.com/gin-gonic/gin"
)

type ChessController interface {
	GetAllChessGame(c *gin.Context)
	GetChessGameById(c *gin.Context)
	CreateChessGame(c *gin.Context)
	JoinChessGame(c *gin.Context)
	MakeMove(c *gin.Context)
}

type ChessControllerImpl struct {
	svc service.ChessService
}

func (u ChessControllerImpl) GetAllChessGame(c *gin.Context) {
	u.svc.GetAllChessGame(c)
}

func (u ChessControllerImpl) GetChessGameById(c *gin.Context) {
	u.svc.GetChessGameById(c)
}

func (u ChessControllerImpl) CreateChessGame(c *gin.Context) {
	u.svc.CreateChessGame(c)
}

func (u ChessControllerImpl) JoinChessGame(c *gin.Context) {
	u.svc.JoinChessGame(c)
}

func (u ChessControllerImpl) MakeMove(c *gin.Context) {
	u.svc.MakeMove(c)
}

func ChessControllerInit(chessService service.ChessService) *ChessControllerImpl {
	return &ChessControllerImpl{
		svc: chessService,
	}
}
