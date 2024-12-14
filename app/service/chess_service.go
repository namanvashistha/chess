package service

import (
	"chess-engine/app/constant"
	"chess-engine/app/domain/dao"
	"chess-engine/app/domain/dto"
	"chess-engine/app/engine"
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
	MakeMove(c *gin.Context)
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

	// Fetch the chess game state from the repository
	gameState, err := u.chessRepository.GetChessGameState("3")
	if err != nil {
		log.Error("Happened error when getting chess game state. Error", err)
		pkg.PanicException(constant.UnknownError)
	}

	// Get the allowed moves based on the game state
	allowedMoves := engine.GetAllowedMoves(gameState)
	boardlayout := engine.GetBoardLayout()
	pieceMap := engine.GetPiecesMap()

	// Build the response with game state and allowed moves
	response := map[string]interface{}{
		"board":         gameState.Board, // assuming `gameState.Board` contains the chessboard state
		"turn":          gameState.Turn,
		"status":        gameState.Status,
		"last_move":     gameState.LastMove,
		"allowed_moves": allowedMoves,
		"board_layout":  boardlayout,
		"pieces_map":    pieceMap,
	}

	// Return the chess game state and allowed moves as the response
	c.JSON(http.StatusOK, pkg.BuildResponse(constant.Success, response))
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

func (u ChessServiceImpl) MakeMove(c *gin.Context) {
	// piece: "wR1",
	// source_square: "a1",
	// destination_square: "a4",

	defer pkg.PanicHandler(c)

	log.Info("start to execute program make move")
	var request dto.Move
	if err := c.ShouldBindJSON(&request); err != nil {
		log.Error("Happened error when mapping request from FE. Error", err)
		pkg.PanicException(constant.InvalidRequest)
	}
	request.GameId = "3"
	game, err := u.chessRepository.GetChessGameState(request.GameId)
	if err != nil {
		log.Error("Happened error when getting chess game state. Error", err)
		pkg.PanicException(constant.UnknownError)
	}

	err = engine.MakeMove(&game, request)
	if err != nil {
		log.Error("Happened error when saving data to database. Error", err)
		pkg.PanicException(constant.UnknownError)
	}

	err = u.chessRepository.SaveChessGame(&game)
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
	// {"square": {"squareColor", "`pieceColor``pieceType``pieceId"}}
	initialBoard := map[string][]string{
		//Rank 8
		"a8": {"w", "bR1"},
		"b8": {"b", "bN1"},
		"c8": {"w", "bB1"},
		"d8": {"b", "bQ1"},
		"e8": {"w", "bK1"},
		"f8": {"b", "bB2"},
		"g8": {"w", "bN2"},
		"h8": {"b", "bR2"},
		//Rank 7
		"a7": {"b", "bP1"},
		"b7": {"w", "bP2"},
		"c7": {"b", "bP3"},
		"d7": {"w", "bP4"},
		"e7": {"b", "bP5"},
		"f7": {"w", "bP6"},
		"g7": {"b", "bP7"},
		"h7": {"w", "bP8"},
		//Rank 6
		"a6": {"w", "---"},
		"b6": {"b", "---"},
		"c6": {"w", "---"},
		"d6": {"b", "---"},
		"e6": {"w", "---"},
		"f6": {"b", "---"},
		"g6": {"w", "---"},
		"h6": {"b", "---"},
		//Rank 5
		"a5": {"b", "---"},
		"b5": {"w", "---"},
		"c5": {"b", "---"},
		"d5": {"w", "---"},
		"e5": {"b", "---"},
		"f5": {"w", "---"},
		"g5": {"b", "---"},
		"h5": {"w", "---"},
		//Rank 4
		"a4": {"w", "---"},
		"b4": {"b", "---"},
		"c4": {"w", "---"},
		"d4": {"b", "---"},
		"e4": {"w", "---"},
		"f4": {"b", "---"},
		"g4": {"w", "---"},
		"h4": {"b", "---"},
		//Rank 3
		"a3": {"b", "---"},
		"b3": {"w", "---"},
		"c3": {"b", "---"},
		"d3": {"w", "---"},
		"e3": {"b", "---"},
		"f3": {"w", "---"},
		"g3": {"b", "---"},
		"h3": {"w", "---"},
		//Rank 2
		"a2": {"w", "wP1"},
		"b2": {"b", "wP2"},
		"c2": {"w", "wP3"},
		"d2": {"b", "wP4"},
		"e2": {"w", "wP5"},
		"f2": {"b", "wP6"},
		"g2": {"w", "wP7"},
		"h2": {"b", "wP8"},
		// Rank 1
		"a1": {"b", "wR1"},
		"b1": {"w", "wN1"},
		"c1": {"b", "wB1"},
		"d1": {"w", "wQ1"},
		"e1": {"b", "wK1"},
		"f1": {"w", "wB2"},
		"g1": {"b", "wN2"},
		"h1": {"w", "wR2"},
	}
	// initialBoard := [8][8]string{
	// 	{{"rR", "rN", "rB", "rQ", "rK", "rB", "rN", "rR"},
	// 	{"rP", "rP", "rP", "rP", "rP", "rP", "rP", "rP"},
	// 	{"", "", "", "", "", "", "", ""},
	// 	{"", "", "", "", "", "", "", ""},
	// 	{"", "", "", "", "", "", "", ""},
	// 	{"", "", "", "", "", "", "", ""},
	// 	{"", "bP", "bP", "bP", "bP", "bP", "bP", "bP"},
	// 	{"", "", "bB", "bQ", "bK", "bB", "bN", "bR"},
	// }
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
