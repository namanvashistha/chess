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
	FindAllChess() ([]dao.Chess, error)
	FindChessById(id int) (dao.Chess, error)
	GetChessGameStateFromCache(gameId string) (dao.ChessGame, error)
	GetChessGameStateFromDB(gameId string) (dao.ChessGame, error)
	SaveChessGameToCache(game *dao.ChessGame) error
	SaveChessGameToDB(game *dao.ChessGame) error
}

type ChessRepositoryImpl struct {
	db          *gorm.DB
	redisClient *pkg.RedisClient
}

func ChessRepositoryInit(db *gorm.DB, redisClient *pkg.RedisClient) *ChessRepositoryImpl {
	db.AutoMigrate(&dao.ChessGame{})
	return &ChessRepositoryImpl{
		db:          db,
		redisClient: redisClient,
	}
}

// Find all chess games from the database
func (r ChessRepositoryImpl) FindAllChess() ([]dao.Chess, error) {
	var chesss []dao.Chess
	err := r.db.Find(&chesss).Error
	if err != nil {
		log.Error("Error finding all chess games:", err)
		return nil, err
	}
	return chesss, nil
}

// Find chess by ID from the database
func (r ChessRepositoryImpl) FindChessById(id int) (dao.Chess, error) {
	var chess dao.Chess
	err := r.db.First(&chess, id).Error
	if err != nil {
		log.Error("Error finding chess by ID:", err)
		return dao.Chess{}, err
	}
	return chess, nil
}

// Fetch the chess game state from cache
func (r ChessRepositoryImpl) GetChessGameStateFromCache(gameId string) (dao.ChessGame, error) {
	var game dao.ChessGame
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
func (r ChessRepositoryImpl) GetChessGameStateFromDB(gameId string) (dao.ChessGame, error) {
	var game dao.ChessGame
	err := r.db.Where("id = ?", gameId).First(&game).Error
	if err != nil {
		log.Error("Error fetching chess game state from DB:", err)
		return game, err
	}
	return game, nil
}

// Save the chess game state to the cache
func (r ChessRepositoryImpl) SaveChessGameToCache(game *dao.ChessGame) error {
	gameStateJSON, err := json.Marshal(game)
	if err != nil {
		log.Error("Error marshalling game state for Redis:", err)
		return err
	}
	log.Info("Saving game state to Redis:", fmt.Sprint(game.ID), string(gameStateJSON))
	err = r.redisClient.Set("game_state:"+fmt.Sprint(game.ID), gameStateJSON, time.Minute*10) // Cache for 10 minutes
	if err != nil {
		log.Error("Error saving game state to Redis:", err)
	}
	return err
}

// Save the chess game state to the database
func (r ChessRepositoryImpl) SaveChessGameToDB(game *dao.ChessGame) error {
	if err := r.db.Save(game).Error; err != nil {
		log.Error("Error saving game state to DB:", err)
		return err
	}
	return nil
}
