package repository

import (
	"chess-engine/app/domain/dao"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type ChessRepository interface {
	FindAllChess() ([]dao.Chess, error)
	FindChessById(id int) (dao.Chess, error)
	GetChessGameState(gameId string) (dao.ChessGame, error)
	SaveChessGame(game *dao.ChessGame) error
}

type ChessRepositoryImpl struct {
	db *gorm.DB
}

func (u ChessRepositoryImpl) FindAllChess() ([]dao.Chess, error) {
	var chesss []dao.Chess

	var err = u.db.Find(&chesss).Error
	if err != nil {
		log.Error("Got an error finding all couples. Error: ", err)
		return nil, err
	}

	return chesss, nil
}

func (u ChessRepositoryImpl) FindChessById(id int) (dao.Chess, error) {
	chess := dao.Chess{
		ID: id,
	}
	err := u.db.First(&chess).Error
	if err != nil {
		log.Error("Got an error when find chess by id. Error: ", err)
		return dao.Chess{}, err
	}
	return chess, nil
}

// Fetches the current game state
func (r ChessRepositoryImpl) GetChessGameState(gameId string) (dao.ChessGame, error) {
	var game dao.ChessGame
	// Assuming we want to get the first ongoing game or the last game
	if err := r.db.Where("status = ?", "ongoing").Last(&game).Error; err != nil {
		log.Println("Error fetching chess game state:", err)
		return game, err
	}
	return game, nil
}

// Saves the current game state to the database (after every move)
func (r ChessRepositoryImpl) SaveChessGame(game *dao.ChessGame) error {
	if err := r.db.Save(game).Error; err != nil {
		log.Println("Error saving chess game state:", err)
		return err
	}
	return nil
}

func ChessRepositoryInit(db *gorm.DB) *ChessRepositoryImpl {
	// db.AutoMigrate(&dao.Chess{})
	db.AutoMigrate(&dao.ChessGame{})
	return &ChessRepositoryImpl{
		db: db,
	}
}
