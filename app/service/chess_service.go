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
	// allowedMoves := engine.GetAllowedMoves(game)
	boardlayout := engine.GetBoardLayout()
	// bitBoard := engine.GetBitBoard()

	// game.ChessState.AllowedMoves = allowedMoves
	game.BoardLayout = boardlayout
	game.CurrentState = engine.ConvertGameStateToMap(game.State)
	game.LegalMoves = engine.ConvertLegalMovesToMap(engine.GenerateLegalMovesForAllPositions(game.State))

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

	initialBoard := map[string]string{
		//Rank 8
		"a8": "bR1",
		"b8": "bN1",
		"c8": "bB1",
		"d8": "bQ1",
		"e8": "bK1",
		"f8": "bB2",
		"g8": "bN2",
		"h8": "bR2",
		//Rank 7
		"a7": "bP1",
		"b7": "bP2",
		"c7": "bP3",
		"d7": "bP4",
		"e7": "bP5",
		"f7": "bP6",
		"g7": "bP7",
		"h7": "bP8",
		//Rank 6
		"a6": "---",
		"b6": "---",
		"c6": "---",
		"d6": "---",
		"e6": "---",
		"f6": "---",
		"g6": "---",
		"h6": "---",
		//Rank 5
		"a5": "---",
		"b5": "---",
		"c5": "---",
		"d5": "---",
		"e5": "---",
		"f5": "---",
		"g5": "---",
		"h5": "---",
		//Rank 4
		"a4": "---",
		"b4": "---",
		"c4": "---",
		"d4": "---",
		"e4": "---",
		"f4": "---",
		"g4": "---",
		"h4": "---",
		//Rank 3
		"a3": "---",
		"b3": "---",
		"c3": "---",
		"d3": "---",
		"e3": "---",
		"f3": "---",
		"g3": "---",
		"h3": "---",
		//Rank 2
		"a2": "wP1",
		"b2": "wP2",
		"c2": "wP3",
		"d2": "wP4",
		"e2": "wP5",
		"f2": "wP6",
		"g2": "wP7",
		"h2": "wP8",
		// Rank 1
		"a1": "wR1",
		"b1": "wN1",
		"c1": "wB1",
		"d1": "wQ1",
		"e1": "wK1",
		"f1": "wB2",
		"g1": "wN2",
		"h1": "wR2",
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

	creatorUser, _ := u.chessRepository.FindUserByToken(request.Token)
	newGame := dao.ChessGame{
		InviteCode: pkg.GenerateRandomString(20),
		Winner:     "",
		// ChessState: initialState,
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
	initialGameState := dao.GameState{
		GameID:         newGame.ID,
		WhiteBitboard:  0xFFFF,
		BlackBitboard:  0xFFFF000000000000,
		PawnBitboard:   0x00FF00000000FF00,
		RookBitboard:   0x8100000000000081,
		KnightBitboard: 0x4200000000000042,
		BishopBitboard: 0x2400000000000024,
		QueenBitboard:  0x0800000000000008,
		KingBitboard:   0x1000000000000010,
		EnPassant:      0,
		CastlingRights: "KQkq",
		Turn:           "w",
	}
	// initialGameState := dao.GameState{
	// 	GameID:         newGame.ID,
	// 	WhiteBitboard:  0x000000000001F20C, // Positions of white pieces
	// 	BlackBitboard:  0x01F003C000000000, // Positions of black pieces
	// 	PawnBitboard:   0x0000000000011200, // Positions of pawns
	// 	RookBitboard:   0x0100000000000080, // Positions of rooks
	// 	KnightBitboard: 0x0000000000000042, // Positions of knights
	// 	BishopBitboard: 0x0000000000000024, // Positions of bishops
	// 	QueenBitboard:  0x0000000000000008, // Position of queens
	// 	KingBitboard:   0x0000000000010000, // Position of kings
	// 	EnPassant:      0x0000000000100000, // Position available for en passant
	// 	CastlingRights: "Kq",               // Castling rights
	// 	Turn:           "w",                // White's turn
	// 	LastMove:       "",                 // No moves yet
	// }
	if err := u.chessRepository.SaveGameStateToDB(&initialGameState); err != nil {
		log.Error("Happened error when saving game state to database. Error", err)
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
	var game dao.ChessGame
	var err error

	// Fetch from cache
	game, err = u.chessRepository.GetChessGameFromCache(gameId)
	if err != nil || game.ID == 0 {
		// Fallback to DB if cache miss
		log.Info("Cache miss. Fetching from database.")
		game, err = u.chessRepository.FindChessGameById(gameId)
		if err != nil {
			log.Error("Error fetching game state:", err)
			pkg.PanicException(constant.DataNotFound)
		}
		// Save to cache after fetching from DB
		_ = u.chessRepository.SaveChessGameToCache(&game)
	}

	// Build response
	// allowedMoves := engine.GetAllowedMoves(game)
	// boardlayout := engine.GetBoardLayout()
	// pieceMap := engine.GetPiecesMap()
	response := map[string]interface{}{
		// "board":         game.ChessState.Board,
		// "turn":          game.ChessState.Turn,
		// "status":        game.ChessState.Status,
		// "last_move":     game.ChessState.LastMove,
		// "allowed_moves": allowedMoves,
		// "board_layout":  boardlayout,
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
	game, err := u.chessRepository.GetChessGameFromCache(gameId)
	if err != nil || game.ID == 0 {
		log.Info("Cache miss. Fetching game state from DB.", gameId)
		game, err = u.chessRepository.FindChessGameById(gameId)
		if err != nil {
			log.Error("Error fetching game state:", err)
			pkg.PanicException(constant.DataNotFound)
		}
	}
	user, err := u.chessRepository.FindUserByToken(request.Token)
	if err != nil {
		log.Error("Error fetching user by token:", err)
		pkg.PanicException(constant.DataNotFound)
	}
	// Apply move
	err = engine.MakeMove(&game, request, user)
	if err != nil {
		log.Error("Error processing move:", err)
		pkg.PanicException(constant.UnknownError)
	}

	// Save to both cache and database
	if saveErr := u.chessRepository.SaveChessGameToDB(&game); saveErr == nil {
		_ = u.chessRepository.SaveChessGameToCache(&game)
	}

	c.JSON(http.StatusOK, pkg.BuildResponse(constant.Success, pkg.Null()))

}

func (u ChessServiceImpl) CreateChessState(c *gin.Context) {
	defer pkg.PanicHandler(c)

	log.Info("start to execute program create chess state")

	// Initialize the chess state
	// {"square": "squareColor", "`pieceColor``pieceType``pieceId"}}
	initialBoard := map[string]string{
		//Rank 8
		"a8": "bR1",
		"b8": "bN1",
		"c8": "bB1",
		"d8": "bQ1",
		"e8": "bK1",
		"f8": "bB2",
		"g8": "bN2",
		"h8": "bR2",
		//Rank 7
		"a7": "bP1",
		"b7": "bP2",
		"c7": "bP3",
		"d7": "bP4",
		"e7": "bP5",
		"f7": "bP6",
		"g7": "bP7",
		"h7": "bP8",
		//Rank 6
		"a6": "---",
		"b6": "---",
		"c6": "---",
		"d6": "---",
		"e6": "---",
		"f6": "---",
		"g6": "---",
		"h6": "---",
		//Rank 5
		"a5": "---",
		"b5": "---",
		"c5": "---",
		"d5": "---",
		"e5": "---",
		"f5": "---",
		"g5": "---",
		"h5": "---",
		//Rank 4
		"a4": "---",
		"b4": "---",
		"c4": "---",
		"d4": "---",
		"e4": "---",
		"f4": "---",
		"g4": "---",
		"h4": "---",
		//Rank 3
		"a3": "---",
		"b3": "---",
		"c3": "---",
		"d3": "---",
		"e3": "---",
		"f3": "---",
		"g3": "---",
		"h3": "---",
		//Rank 2
		"a2": "wP1",
		"b2": "wP2",
		"c2": "wP3",
		"d2": "wP4",
		"e2": "wP5",
		"f2": "wP6",
		"g2": "wP7",
		"h2": "wP8",
		// Rank 1
		"a1": "wR1",
		"b1": "wN1",
		"c1": "wB1",
		"d1": "wQ1",
		"e1": "wK1",
		"f1": "wB2",
		"g1": "wN2",
		"h1": "wR2",
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
	// u.chessRepository.SaveChessGameToCache(&initialState)
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
