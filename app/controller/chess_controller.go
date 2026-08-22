package controller

import (
	"chess-engine/app/constant"
	"chess-engine/app/domain/dto"
	"chess-engine/app/pkg"
	"chess-engine/app/service"

	"github.com/gin-gonic/gin"
)

type ChessController interface {
	GetAllChessGame(c *gin.Context)
	GetChessGameById(c *gin.Context)
	CreateChessGame(c *gin.Context)
	CreateBotChessGame(c *gin.Context)
	CreateLocalChessGame(c *gin.Context)
	JoinChessGame(c *gin.Context)
}

type ChessControllerImpl struct {
	svc service.ChessService
}

// bindJSON decodes the request body, reporting a client error rather than a
// server one when it does not parse.
func bindJSON(c *gin.Context, target interface{}) error {
	if err := c.ShouldBindJSON(target); err != nil {
		return pkg.NewError(constant.InvalidRequest, "invalid request body", err)
	}
	return nil
}

func (u ChessControllerImpl) GetAllChessGame(c *gin.Context) {
	data, err := u.svc.GetAllChessGame()
	respond(c, data, err)
}

func (u ChessControllerImpl) GetChessGameById(c *gin.Context) {
	data, err := u.svc.GetChessGameById(c.Param("gameId"))
	respond(c, data, err)
}

func (u ChessControllerImpl) CreateChessGame(c *gin.Context) {
	var request dto.TokenGetRequest
	if err := bindJSON(c, &request); err != nil {
		respond(c, nil, err)
		return
	}
	id, err := u.svc.CreateChessGame(request.Token)
	respond(c, id, err)
}

func (u ChessControllerImpl) CreateLocalChessGame(c *gin.Context) {
	var request dto.TokenGetRequest
	if err := bindJSON(c, &request); err != nil {
		respond(c, nil, err)
		return
	}
	id, err := u.svc.CreateLocalChessGame(request.Token)
	respond(c, id, err)
}

func (u ChessControllerImpl) CreateBotChessGame(c *gin.Context) {
	var request dto.CreateBotGameRequest
	if err := bindJSON(c, &request); err != nil {
		respond(c, nil, err)
		return
	}
	id, err := u.svc.CreateBotChessGame(request.Token, request.Difficulty)
	respond(c, id, err)
}

func (u ChessControllerImpl) JoinChessGame(c *gin.Context) {
	var request dto.JoinChessGameRequest
	if err := bindJSON(c, &request); err != nil {
		respond(c, nil, err)
		return
	}
	id, err := u.svc.JoinChessGame(request.InviteCode, request.Token)
	respond(c, id, err)
}

func ChessControllerInit(chessService service.ChessService) *ChessControllerImpl {
	return &ChessControllerImpl{svc: chessService}
}
