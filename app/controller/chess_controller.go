package controller

import (
	"chess-engine/app/service"

	"github.com/gin-gonic/gin"
)

type ChessController interface {
	GetAllChess(c *gin.Context)
	GetChessById(c *gin.Context)
	GetChessState(c *gin.Context)
	SaveChessGame(c *gin.Context)
	CreateChessState(c *gin.Context)
}

type ChessControllerImpl struct {
	svc service.ChessService
}

func (u ChessControllerImpl) GetAllChess(c *gin.Context) {
	u.svc.GetAllChess(c)
}

func (u ChessControllerImpl) GetChessById(c *gin.Context) {
	u.svc.GetChessById(c)
}

func (u ChessControllerImpl) GetChessState(c *gin.Context) {
	u.svc.GetChessState(c)
}

func (u ChessControllerImpl) SaveChessGame(c *gin.Context) {
	u.svc.SaveChessGame(c)
}

func (u ChessControllerImpl) CreateChessState(c *gin.Context) {
	u.svc.CreateChessState(c)
}

func ChessControllerInit(chessService service.ChessService) *ChessControllerImpl {
	return &ChessControllerImpl{
		svc: chessService,
	}
}
