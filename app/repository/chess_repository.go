package repository

import (
	"chess-engine/app/domain/dao"
	"chess-engine/app/pkg"
	"encoding/json"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type ChessRepository interface {
	FindAllChessGame() ([]dao.ChessGame, error)
	FindChessGameById(id string) (dao.ChessGame, error)
	SaveChessGameToDB(game *dao.ChessGame) error
	GetChessStateStateFromCache(gameId string) (dao.ChessState, error)
	GetChessStateStateFromDB(gameId string) (dao.ChessState, error)
	SaveChessStateToCache(game *dao.ChessState) error
	SaveChessStateToDB(game *dao.ChessState) error
	FindUserByToken(token string) (dao.User, error)
}

type ChessRepositoryImpl struct {
	db          *gorm.DB
	redisClient *pkg.RedisClient
}

func ChessRepositoryInit(db *gorm.DB, redisClient *pkg.RedisClient) *ChessRepositoryImpl {
	db.AutoMigrate(&dao.ChessState{})
	db.AutoMigrate(&dao.ChessGame{})
	return &ChessRepositoryImpl{
		db:          db,
		redisClient: redisClient,
	}
}

// Find all chess games from the database
func (r ChessRepositoryImpl) FindAllChessGame() ([]dao.ChessGame, error) {
	var chesses []dao.ChessGame
	err := r.db.
		Preload("WhiteUser", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name")
		}).
		Preload("BlackUser", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name")
		}).Order("id desc").Find(&chesses).Error
	if err != nil {
		log.Error("Error finding all chess games:", err)
		return nil, err
	}
	return chesses, nil
}

// Find chess by ID from the database
func (r ChessRepositoryImpl) FindChessGameById(id string) (dao.ChessGame, error) {
	var chess dao.ChessGame
	err := r.db.Preload("ChessState").First(&chess, id).Error
	if err != nil {
		log.Error("Error finding chess by ID:", err)
		return dao.ChessGame{}, err
	}
	return chess, nil
}

func (r ChessRepositoryImpl) SaveChessGameToDB(game *dao.ChessGame) error {
	if err := r.db.Save(game).Error; err != nil {
		log.Error("Error saving chess game to DB:", err)
		return err
	}
	return nil
}

// Fetch the chess game state from cache
func (r ChessRepositoryImpl) GetChessStateStateFromCache(gameId string) (dao.ChessState, error) {
	var game dao.ChessState
	cachedState, err := r.redisClient.Get("game_state:" + gameId)
	if err != nil || cachedState == "" {
		return game, err
	}
	err = json.Unmarshal([]byte(cachedState), &game)
	if err != nil {
		log.Error("Error unmarshalling game state from Redis:", err)
		return game, err
	}
	return game, nil
}

// Fetch the chess game state from the database
func (r ChessRepositoryImpl) GetChessStateStateFromDB(gameId string) (dao.ChessState, error) {
	var game dao.ChessState
	err := r.db.Where("id = ?", gameId).First(&game).Error
	if err != nil {
		log.Error("Error fetching chess game state from DB:", err)
		return game, err
	}
	return game, nil
}

// Save the chess game state to the cache
func (r ChessRepositoryImpl) SaveChessStateToCache(game *dao.ChessState) error {
	gameStateJSON, err := json.Marshal(game)
	if err != nil {
		log.Error("Error marshalling game state for Redis:", err)
		return err
	}
	err = r.redisClient.Set("game_state:"+fmt.Sprint(game.ID), gameStateJSON, time.Minute*10) // Cache for 10 minutes
	if err != nil {
		log.Error("Error saving game state to Redis:", err)
	}
	return err
}

// Save the chess game state to the database
func (r ChessRepositoryImpl) SaveChessStateToDB(game *dao.ChessState) error {
	if err := r.db.Save(game).Error; err != nil {
		log.Error("Error saving game state to DB:", err)
		return err
	}
	return nil
}

func (u ChessRepositoryImpl) FindUserByToken(token string) (dao.User, error) {
	var user dao.User
	err := u.db.Where("token = ?", token).First(&user).Error
	if err != nil {
		log.Error("Got and error when find user by token. Error: ", err)
		return dao.User{}, err
	}
	return user, nil
}
