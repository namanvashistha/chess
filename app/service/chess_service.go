package service

import (
	"chess-engine/app/constant"
	"chess-engine/app/domain/dao"
	"chess-engine/app/engine"
	"chess-engine/app/pkg"
	"chess-engine/app/repository"

	log "github.com/sirupsen/logrus"
)

// ChessService is transport-agnostic: plain arguments in, (value, error) out.
type ChessService interface {
	GetAllChessGame() ([]dao.ChessGame, error)
	GetChessGameById(gameID string) (dao.ChessGame, error)
	CreateChessGame(token string) (int, error)
	CreateBotChessGame(token, difficulty string) (int, error)
	CreateLocalChessGame(token string) (int, error)
	JoinChessGame(inviteCode, token string) (int, error)
}

type ChessServiceImpl struct {
	chessRepository repository.ChessRepository
}

func (u ChessServiceImpl) GetChessGameById(gameID string) (dao.ChessGame, error) {
	game, err := u.chessRepository.FindChessGameById(gameID)
	if err != nil {
		return dao.ChessGame{}, pkg.NewError(constant.DataNotFound, "game not found", err)
	}

	game.BoardLayout = engine.GetBoardLayout()
	game.CurrentState = engine.ConvertGameStateToMap(game.State)
	legalMoves, _ := engine.GenerateLegalMovesForAllPositions(game.State)
	game.LegalMoves = engine.ConvertLegalMovesToMap(engine.FilterMovesByTurn(legalMoves, game.State))
	return game, nil
}

func (u ChessServiceImpl) GetAllChessGame() ([]dao.ChessGame, error) {
	data, err := u.chessRepository.FindAllChessGame()
	if err != nil {
		return nil, pkg.NewError(constant.UnknownError, "could not list games", err)
	}
	return data, nil
}

// resolveUser looks up the caller by session token.
func (u ChessServiceImpl) resolveUser(token string) (dao.User, error) {
	user, err := u.chessRepository.FindUserByToken(token)
	if err != nil || user.ID == 0 {
		return dao.User{}, pkg.NewError(constant.Unauthorized, "session expired, please reload", err)
	}
	return user, nil
}

// persistNewGame saves a game and its opening position.
//
// The starting position comes from engine.StartState() rather than a bitboard
// literal. The literal was pasted into all three game-creation paths, which is
// three places to update and three chances for them to drift from what the
// engine itself considers the start.
func (u ChessServiceImpl) persistNewGame(game *dao.ChessGame) (int, error) {
	if err := u.chessRepository.SaveChessGameToDB(game); err != nil {
		return 0, pkg.NewError(constant.UnknownError, "could not create game", err)
	}

	state := engine.StartState()
	state.GameID = game.ID
	if err := u.chessRepository.SaveGameStateToDB(&state); err != nil {
		return 0, pkg.NewError(constant.UnknownError, "could not create game state", err)
	}
	return game.ID, nil
}

func (u ChessServiceImpl) CreateChessGame(token string) (int, error) {
	creator, err := u.resolveUser(token)
	if err != nil {
		return 0, err
	}

	game := dao.ChessGame{InviteCode: pkg.GenerateRandomString(20)}
	if pkg.GenerateRandomBool() {
		game.WhiteUser = &creator
	} else {
		game.BlackUser = &creator
	}
	return u.persistNewGame(&game)
}

// CreateLocalChessGame creates a pass-and-play game with the human seated as
// BOTH colours, so a single client controls both sides on one device. The
// server's turn check passes for either side because the same user owns both.
func (u ChessServiceImpl) CreateLocalChessGame(token string) (int, error) {
	human, err := u.resolveUser(token)
	if err != nil {
		return 0, err
	}

	game := dao.ChessGame{
		InviteCode: pkg.GenerateRandomString(20),
		WhiteUser:  &human,
		BlackUser:  &human,
	}
	return u.persistNewGame(&game)
}

// CreateBotChessGame seats the human (always White) against the built-in bot.
// Both seats are filled so play starts immediately.
func (u ChessServiceImpl) CreateBotChessGame(token, difficulty string) (int, error) {
	level := difficulty
	if level != "easy" && level != "medium" && level != "hard" {
		level = "easy"
	}

	human, err := u.resolveUser(token)
	if err != nil {
		return 0, err
	}
	bot, err := u.chessRepository.FindOrCreateBotUser()
	if err != nil {
		return 0, pkg.NewError(constant.UnknownError, "could not resolve the bot player", err)
	}

	game := dao.ChessGame{
		InviteCode: pkg.GenerateRandomString(20),
		BotLevel:   level,
		WhiteUser:  &human,
		BlackUser:  &bot,
	}
	return u.persistNewGame(&game)
}

func (u ChessServiceImpl) JoinChessGame(inviteCode, token string) (int, error) {
	game, err := u.chessRepository.FindChessGameByInviteCode(inviteCode)
	if err != nil {
		return 0, pkg.NewError(constant.DataNotFound, "no game with that invite code", err)
	}

	joiner, err := u.resolveUser(token)
	if err != nil {
		return 0, err
	}

	switch {
	case game.WhiteUser == nil:
		game.WhiteUser = &joiner
	case game.BlackUser == nil:
		game.BlackUser = &joiner
	default:
		return 0, pkg.NewError(constant.InvalidRequest, "game is full", nil)
	}

	if err := u.chessRepository.SaveChessGameToDB(&game); err != nil {
		log.Error("Happened error when saving game to database. Error", err)
		return 0, pkg.NewError(constant.UnknownError, "could not join game", err)
	}
	return game.ID, nil
}

func ChessServiceInit(chessRepository repository.ChessRepository) *ChessServiceImpl {
	return &ChessServiceImpl{chessRepository: chessRepository}
}
