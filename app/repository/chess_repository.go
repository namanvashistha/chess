package repository

import (
	"chess-engine/app/domain/dao"
	"chess-engine/app/pkg"
	"encoding/json"
	"time"

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
	db          *gorm.DB
	redisClient *pkg.RedisClient
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

func ChessRepositoryInit(db *gorm.DB, redisClient *pkg.RedisClient) *ChessRepositoryImpl {
	// db.AutoMigrate(&dao.Chess{})
	db.AutoMigrate(&dao.ChessGame{})
	return &ChessRepositoryImpl{
		db:          db,
		redisClient: redisClient,
	}
}

func (r ChessRepositoryImpl) GetChessGameState(gameId string) (dao.ChessGame, error) {
	var game dao.ChessGame

	// First, check Redis cache
	cachedState, err := r.redisClient.Get("game_state:" + gameId)
	if err != nil {
		return game, err
	}

	if cachedState != "" {
		// Deserialize the cached state (e.g., from JSON or other format)
		err = json.Unmarshal([]byte(cachedState), &game)
		if err != nil {
			log.Error("Failed to unmarshal game state from Redis:", err)
			return game, err
		}
		return game, nil
	}

	// If not in cache, fetch from the database
	if err := r.db.Where("id = ?", gameId).First(&game).Error; err != nil {
		log.Error("Error fetching chess game state from DB:", err)
		return game, err
	}

	// Cache the game state in Redis for future use
	gameStateJSON, _ := json.Marshal(game)
	_ = r.redisClient.Set("game_state:"+gameId, gameStateJSON, time.Minute*10) // Cache for 10 minutes

	return game, nil
}

func (r ChessRepositoryImpl) SaveChessGame(game *dao.ChessGame) error {
	// Save to database
	if err := r.db.Save(game).Error; err != nil {
		log.Error("Error saving chess game state to DB:", err)
		return err
	}

	// Update cache
	gameStateJSON, _ := json.Marshal(game)
	_ = r.redisClient.Set("game_state:"+string(game.ID), gameStateJSON, time.Minute*10)

	return nil
}
