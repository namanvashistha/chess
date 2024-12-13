package service

import (
	"chess-engine/app/constant"
	"chess-engine/app/domain/dao"
	"chess-engine/app/pkg"
	"chess-engine/app/repository"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type ChessService interface {
	GetAllChess(c *gin.Context)
	GetChessById(c *gin.Context)
	GetChessState(c *gin.Context)
	SaveChessGame(c *gin.Context)
	CreateChessState(c *gin.Context)
}

type ChessServiceImpl struct {
	chessRepository repository.ChessRepository
}

func (u ChessServiceImpl) GetChessById(c *gin.Context) {
	defer pkg.PanicHandler(c)

	log.Info("start to execute program get chess by id")
	chessID, _ := strconv.Atoi(c.Param("chessID"))

	data, err := u.chessRepository.FindChessById(chessID)
	if err != nil {
		log.Error("Happened error when get data from database. Error", err)
		pkg.PanicException(constant.DataNotFound)
	}

	c.JSON(http.StatusOK, pkg.BuildResponse(constant.Success, data))
}

func (u ChessServiceImpl) GetAllChess(c *gin.Context) {
	defer pkg.PanicHandler(c)

	log.Info("start to execute get all data chess")

	data, err := u.chessRepository.FindAllChess()
	if err != nil {
		log.Error("Happened Error when find all chess data. Error: ", err)
		pkg.PanicException(constant.UnknownError)
	}

	c.JSON(http.StatusOK, pkg.BuildResponse(constant.Success, data))
}

func (u ChessServiceImpl) GetChessState(c *gin.Context) {
	defer pkg.PanicHandler(c)

	log.Info("start to execute get chess state")

	gameState, err := u.chessRepository.GetChessGameState()
	if err != nil {
		log.Error("Happened error when getting chess game state. Error", err)
		pkg.PanicException(constant.UnknownError)
	}

	// Return the chess game state as the response
	c.JSON(http.StatusOK, pkg.BuildResponse(constant.Success, gameState))
}

func (u ChessServiceImpl) SaveChessGame(c *gin.Context) {
	defer pkg.PanicHandler(c)

	log.Info("start to execute program save chess game")
	var request dao.ChessGame
	if err := c.ShouldBindJSON(&request); err != nil {
		log.Error("Happened error when mapping request from FE. Error", err)
		pkg.PanicException(constant.InvalidRequest)
	}

	err := u.chessRepository.SaveChessGame(&request)
	if err != nil {
		log.Error("Happened error when saving data to database. Error", err)
		pkg.PanicException(constant.UnknownError)
	}

	c.JSON(http.StatusOK, pkg.BuildResponse(constant.Success, pkg.Null()))
}

func (u ChessServiceImpl) CreateChessState(c *gin.Context) {
	defer pkg.PanicHandler(c)

	log.Info("start to execute program create chess state")

	// Initialize the chess state
	initialBoard := [8][8]string{
		{"rR", "rN", "rB", "rQ", "rK", "rB", "rN", "rR"},
		{"rP", "rP", "rP", "rP", "rP", "rP", "rP", "rP"},
		{"", "", "", "", "", "", "", ""},
		{"", "", "", "", "", "", "", ""},
		{"", "", "", "", "", "", "", ""},
		{"", "", "", "", "", "", "", ""},
		{"bP", "bP", "bP", "bP", "bP", "bP", "bP", "bP"},
		{"bR", "bN", "bB", "bQ", "bK", "bB", "bN", "bR"},
	}
	boardJSON, err := json.Marshal(initialBoard)
	if err != nil {
		log.Error("Error converting board to JSON. Error", err)
		pkg.PanicException(constant.UnknownError)
	}
	// Create a new ChessGame instance
	initialState := dao.ChessGame{
		Board:    boardJSON,
		Turn:     "white",   // Starting with white
		Status:   "ongoing", // Initial game status
		LastMove: "",        // No moves yet
	}

	// Save the initial chess state to the database
	err = u.chessRepository.SaveChessGame(&initialState)
	if err != nil {
		log.Error("Happened error when saving data to database. Error", err)
		pkg.PanicException(constant.UnknownError)
	}

	// Return a success response
	c.JSON(http.StatusOK, pkg.BuildResponse(constant.Success, pkg.Null()))
}

func ChessServiceInit(chessRepository repository.ChessRepository) *ChessServiceImpl {
	return &ChessServiceImpl{
		chessRepository: chessRepository,
	}
}
