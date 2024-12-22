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

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type ChessService interface {
	GetAllChessGame(c *gin.Context)
	GetChessGameById(c *gin.Context)
	CreateChessGame(c *gin.Context)
	JoinChessGame(c *gin.Context)
	GetChessState(c *gin.Context)
	SaveChessState(c *gin.Context)
	CreateChessState(c *gin.Context)
	MakeMove(c *gin.Context)
}

type ChessServiceImpl struct {
	chessRepository repository.ChessRepository
}

func (u ChessServiceImpl) GetChessGameById(c *gin.Context) {
	defer pkg.PanicHandler(c)

	log.Info("start to execute program get chess by id")
	gameId := c.Param("gameId")
	log.Info(gameId)
	game, err := u.chessRepository.FindChessGameById(gameId)
	if err != nil {
		log.Error("Happened error when get data from database. Error", err)
		pkg.PanicException(constant.DataNotFound)
	}
	allowedMoves := engine.GetAllowedMoves(game.ChessState)
	boardlayout := engine.GetBoardLayout()

	game.ChessState.AllowedMoves = allowedMoves
	game.ChessState.BoardLayout = boardlayout

	c.JSON(http.StatusOK, pkg.BuildResponse(constant.Success, game))
}

func (u ChessServiceImpl) GetAllChessGame(c *gin.Context) {
	defer pkg.PanicHandler(c)

	log.Info("start to execute get all data chess")

	data, err := u.chessRepository.FindAllChessGame()
	if err != nil {
		log.Error("Happened Error when find all chess data. Error: ", err)
		pkg.PanicException(constant.UnknownError)
	}

	c.JSON(http.StatusOK, pkg.BuildResponse(constant.Success, data))
}

func (u ChessServiceImpl) CreateChessGame(c *gin.Context) {
	defer pkg.PanicHandler(c)

	log.Info("start to execute program create chess state")
	var request dto.TokenGetRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		log.Error("Happened error when mapping request from FE. Error", err)
		pkg.PanicException(constant.InvalidRequest)
	}
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
	boardJSON, err := json.Marshal(initialBoard)
	if err != nil {
		log.Error("Error converting board to JSON. Error", err)
		pkg.PanicException(constant.UnknownError)
	}
	// Create a new ChessState instance
	initialState := dao.ChessState{
		Board:    boardJSON,
		Turn:     "white",
		Status:   "ongoing",
		LastMove: "",
	}
	log.Info("start to execute program create chess game", request.Token)
	creatorUser, err := u.chessRepository.FindUserByToken(request.Token)
	newGame := dao.ChessGame{
		InviteCode: pkg.GenerateRandomString(20),
		Winner:     "",
		ChessState: initialState,
	}

	if isAssignWhite := pkg.GenerateRandomBool(); isAssignWhite {
		newGame.WhiteUser = &creatorUser
	} else {
		newGame.BlackUser = &creatorUser
	}

	if err := u.chessRepository.SaveChessStateToDB(&initialState); err != nil {
		log.Error("Happened error when saving state to database. Error", err)
		pkg.PanicException(constant.UnknownError)
	}
	if err := u.chessRepository.SaveChessGameToDB(&newGame); err != nil {
		log.Error("Happened error when saving game to database. Error", err)
		pkg.PanicException(constant.UnknownError)
	}

	// Return a success response
	c.JSON(http.StatusOK, pkg.BuildResponse(constant.Success, newGame.ID))

}

func (u ChessServiceImpl) JoinChessGame(c *gin.Context) {
	defer pkg.PanicHandler(c)

	log.Info("start to execute program join chess game")
	var request dto.JoinChessGameRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		log.Error("Happened error when mapping request from FE. Error", err)
		pkg.PanicException(constant.InvalidRequest)
	}

	game, err := u.chessRepository.FindChessGameByInviteCode(request.InviteCode)
	if err != nil {
		log.Error("Happened error when get data from database. Error", err)
		pkg.PanicException(constant.DataNotFound)
	}

	joinUser, err := u.chessRepository.FindUserByToken(request.Token)
	if err != nil {
		log.Error("Happened error when get data from database. Error", err)
		pkg.PanicException(constant.DataNotFound)
	}

	if game.WhiteUser == nil {
		game.WhiteUser = &joinUser
	} else if game.BlackUser == nil {
		game.BlackUser = &joinUser
	} else {
		log.Error("Game is full. Cannot join.")
		pkg.PanicException(constant.InvalidRequest)
	}

	if err := u.chessRepository.SaveChessGameToDB(&game); err != nil {
		log.Error("Happened error when saving game to database. Error", err)
		pkg.PanicException(constant.UnknownError)
	}

	c.JSON(http.StatusOK, pkg.BuildResponse(constant.Success, game.ID))
}

func (u ChessServiceImpl) GetChessState(c *gin.Context) {
	defer pkg.PanicHandler(c)

	log.Info("Fetching chess game state")

	gameId := c.Param("gameId")
	log.Info(gameId)
	var game dao.ChessState
	var err error

	// Fetch from cache
	game, err = u.chessRepository.GetChessStateStateFromCache(gameId)
	if err != nil || game.ID == 0 {
		// Fallback to DB if cache miss
		log.Info("Cache miss. Fetching from database.")
		game, err = u.chessRepository.GetChessStateStateFromDB(gameId)
		if err != nil {
			log.Error("Error fetching game state:", err)
			pkg.PanicException(constant.DataNotFound)
		}
		// Save to cache after fetching from DB
		_ = u.chessRepository.SaveChessStateToCache(&game)
	}

	// Build response
	allowedMoves := engine.GetAllowedMoves(game)
	boardlayout := engine.GetBoardLayout()
	// pieceMap := engine.GetPiecesMap()
	response := map[string]interface{}{
		"board":         game.Board,
		"turn":          game.Turn,
		"status":        game.Status,
		"last_move":     game.LastMove,
		"allowed_moves": allowedMoves,
		"board_layout":  boardlayout,
		// "pieces_map":    pieceMap,
	}

	c.JSON(http.StatusOK, pkg.BuildResponse(constant.Success, response))
}

func (u ChessServiceImpl) SaveChessState(c *gin.Context) {
	defer pkg.PanicHandler(c)

	log.Info("start to execute program save chess game")
	var request dao.ChessState
	if err := c.ShouldBindJSON(&request); err != nil {
		log.Error("Happened error when mapping request from FE. Error", err)
		pkg.PanicException(constant.InvalidRequest)
	}

	err := u.chessRepository.SaveChessStateToDB(&request)
	if err != nil {
		log.Error("Happened error when saving data to database. Error", err)
		pkg.PanicException(constant.UnknownError)
	}

	c.JSON(http.StatusOK, pkg.BuildResponse(constant.Success, pkg.Null()))
}

func (u ChessServiceImpl) MakeMove(c *gin.Context) {
	defer pkg.PanicHandler(c)

	log.Info("Processing move")
	var request dto.Move
	if err := c.ShouldBindJSON(&request); err != nil {
		log.Error("Invalid move request:", err)
		pkg.PanicException(constant.InvalidRequest)
	}

	// Fetch the current game state
	gameId := request.GameId
	game, err := u.chessRepository.GetChessStateStateFromCache(gameId)
	if err != nil || game.ID == 0 {
		log.Info("Cache miss. Fetching game state from DB.", gameId)
		game, err = u.chessRepository.GetChessStateStateFromDB(gameId)
		if err != nil {
			log.Error("Error fetching game state:", err)
			pkg.PanicException(constant.DataNotFound)
		}
	}

	// Apply move
	err = engine.MakeMove(&game, request)
	if err != nil {
		log.Error("Error processing move:", err)
		pkg.PanicException(constant.UnknownError)
	}

	// Save to both cache and database
	if saveErr := u.chessRepository.SaveChessStateToDB(&game); saveErr == nil {
		_ = u.chessRepository.SaveChessStateToCache(&game)
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
	boardJSON, err := json.Marshal(initialBoard)
	if err != nil {
		log.Error("Error converting board to JSON. Error", err)
		pkg.PanicException(constant.UnknownError)
	}
	// Create a new ChessState instance
	initialState := dao.ChessState{
		Board:    boardJSON,
		Turn:     "white",   // Starting with white
		Status:   "ongoing", // Initial game status
		LastMove: "",        // No moves yet
	}

	// Save the initial chess state to the cache and db
	err = u.chessRepository.SaveChessStateToDB(&initialState)
	u.chessRepository.SaveChessStateToCache(&initialState)
	if err != nil {
		log.Error("Happened error when saving data to database. Error", err)
		pkg.PanicException(constant.UnknownError)
	}

	// Return a success response
	c.JSON(http.StatusOK, pkg.BuildResponse(constant.Success, initialState.ID))
}

func ChessServiceInit(chessRepository repository.ChessRepository) *ChessServiceImpl {
	return &ChessServiceImpl{
		chessRepository: chessRepository,
	}
}
