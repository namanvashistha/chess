package service

import (
	"chess-engine/app/constant"
	"chess-engine/app/domain/dao"
	"chess-engine/app/domain/dto"
	"chess-engine/app/engine"
	"chess-engine/app/pkg"
	"chess-engine/app/repository"
	"net/http"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type ChessService interface {
	GetAllChessGame(c *gin.Context)
	GetChessGameById(c *gin.Context)
	CreateChessGame(c *gin.Context)
	CreateBotChessGame(c *gin.Context)
	CreateLocalChessGame(c *gin.Context)
	JoinChessGame(c *gin.Context)
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
	boardlayout := engine.GetBoardLayout()

	game.BoardLayout = boardlayout
	game.CurrentState = engine.ConvertGameStateToMap(game.State)
	legalMoves, _ := engine.GenerateLegalMovesForAllPositions(game.State)
	legalMoves = engine.FilterMovesByTurn(legalMoves, game.State)
	game.LegalMoves = engine.ConvertLegalMovesToMap(legalMoves)

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

	creatorUser, _ := u.chessRepository.FindUserByToken(request.Token)
	newGame := dao.ChessGame{
		InviteCode: pkg.GenerateRandomString(20),
		Winner:     "",
	}

	if isAssignWhite := pkg.GenerateRandomBool(); isAssignWhite {
		newGame.WhiteUser = &creatorUser
	} else {
		newGame.BlackUser = &creatorUser
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
	if err := u.chessRepository.SaveGameStateToDB(&initialGameState); err != nil {
		log.Error("Happened error when saving game state to database. Error", err)
		pkg.PanicException(constant.UnknownError)
	}

	// Return a success response
	c.JSON(http.StatusOK, pkg.BuildResponse(constant.Success, newGame.ID))

}

// CreateLocalChessGame creates a pass-and-play game with the human seated as
// BOTH colours, so a single client controls both sides on one device. The
// server's turn check passes for either side because the same user owns both.
func (u ChessServiceImpl) CreateLocalChessGame(c *gin.Context) {
	defer pkg.PanicHandler(c)

	log.Info("start to execute program create local (pass & play) chess game")
	var request dto.TokenGetRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		log.Error("Happened error when mapping request from FE. Error", err)
		pkg.PanicException(constant.InvalidRequest)
	}

	human, err := u.chessRepository.FindUserByToken(request.Token)
	if err != nil {
		log.Error("Happened error when finding user by token. Error", err)
		pkg.PanicException(constant.DataNotFound)
	}

	newGame := dao.ChessGame{
		InviteCode: pkg.GenerateRandomString(20),
		Winner:     "",
		WhiteUser:  &human,
		BlackUser:  &human,
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
	if err := u.chessRepository.SaveGameStateToDB(&initialGameState); err != nil {
		log.Error("Happened error when saving game state to database. Error", err)
		pkg.PanicException(constant.UnknownError)
	}

	c.JSON(http.StatusOK, pkg.BuildResponse(constant.Success, newGame.ID))
}

// CreateBotChessGame creates a game seating the human (always White) against the
// built-in bot. Both seats are filled so play starts immediately.
func (u ChessServiceImpl) CreateBotChessGame(c *gin.Context) {
	defer pkg.PanicHandler(c)

	log.Info("start to execute program create chess game vs bot")
	var request dto.CreateBotGameRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		log.Error("Happened error when mapping request from FE. Error", err)
		pkg.PanicException(constant.InvalidRequest)
	}

	level := request.Difficulty
	if level != "easy" && level != "medium" && level != "hard" {
		level = "easy"
	}

	human, err := u.chessRepository.FindUserByToken(request.Token)
	if err != nil {
		log.Error("Happened error when finding user by token. Error", err)
		pkg.PanicException(constant.DataNotFound)
	}
	bot, err := u.chessRepository.FindOrCreateBotUser()
	if err != nil {
		log.Error("Happened error when resolving bot user. Error", err)
		pkg.PanicException(constant.UnknownError)
	}

	newGame := dao.ChessGame{
		InviteCode: pkg.GenerateRandomString(20),
		Winner:     "",
		BotLevel:   level,
	}
	// The human always plays White against the bot.
	newGame.WhiteUser = &human
	newGame.BlackUser = &bot

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
	if err := u.chessRepository.SaveGameStateToDB(&initialGameState); err != nil {
		log.Error("Happened error when saving game state to database. Error", err)
		pkg.PanicException(constant.UnknownError)
	}

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

func ChessServiceInit(chessRepository repository.ChessRepository) *ChessServiceImpl {
	return &ChessServiceImpl{
		chessRepository: chessRepository,
	}
}
