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
	FindChessGameByInviteCode(inviteCode string) (dao.ChessGame, error)
	GetChessGameFromCache(gameId string) (dao.ChessGame, error)
	// GetChessGameFromDB(gameId string) (dao.ChessGame, error)
	SaveChessGameToCache(game *dao.ChessGame) error
	SaveGameStateToDB(game *dao.GameState) error
	SaveChessGameToDB(game *dao.ChessGame) error
	FindUserByToken(token string) (dao.User, error)
	SaveGameMoveToDB(game *dao.GameMove) error
}

type ChessRepositoryImpl struct {
	db          *gorm.DB
	redisClient *pkg.RedisClient
}

func ChessRepositoryInit(db *gorm.DB, redisClient *pkg.RedisClient) *ChessRepositoryImpl {
	db.AutoMigrate(&dao.ChessGame{})
	db.AutoMigrate(&dao.GameState{})
	db.AutoMigrate(&dao.GameMove{})
	// gorm.RegisterSerializer("bitboard", serializer.BitboardSerializer{})
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
	err := r.db.
		Preload("WhiteUser", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name")
		}).
		Preload("BlackUser", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name")
		}).
		Preload("State").Preload("Moves").
		First(&chess, id).Error
	if err != nil {
		log.Error("Error finding chess by ID:", err)
		return dao.ChessGame{}, err
	}
	return chess, nil
}

// Find chess game by invite code from the database
func (r ChessRepositoryImpl) FindChessGameByInviteCode(inviteCode string) (dao.ChessGame, error) {
	var chess dao.ChessGame
	err := r.db.
		Preload("WhiteUser", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name")
		}).
		Preload("BlackUser", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name")
		}).First(&chess, "invite_code = ?", inviteCode).Error
	if err != nil {
		log.Error("Error finding chess by invite code:", err)
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
func (r ChessRepositoryImpl) GetChessGameFromCache(gameId string) (dao.ChessGame, error) {
	var game dao.ChessGame
	cachedState, err := r.redisClient.Get("chess_game:" + gameId)
	if err != nil || cachedState == "" {
		return game, err
	}
	err = json.Unmarshal([]byte(cachedState), &game)
	if err != nil {
		log.Error("Error unmarshalling game from Redis:", err)
		return game, err
	}
	return game, nil
}

// Fetch the chess game state from the database
// func (r ChessRepositoryImpl) GetChessGameFromDB(gameId string) (dao.ChessGame, error) {
// 	var game dao.ChessGame
// 	err := r.db.
// 		Preload("WhiteUser", func(db *gorm.DB) *gorm.DB {
// 			return db.Select("id, name")
// 		}).
// 		Preload("BlackUser", func(db *gorm.DB) *gorm.DB {
// 			return db.Select("id, name")
// 		}).
// 		Preload("State").First(&game, id).Error
// 	if err != nil {
// 		log.Error("Error fetching chess game state from DB:", err)
// 		return game, err
// 	}
// 	return game, nil
// }

// Save the chess game state to the cache
func (r ChessRepositoryImpl) SaveChessGameToCache(game *dao.ChessGame) error {
	gameJSON, err := json.Marshal(game)
	if err != nil {
		log.Error("Error marshalling game for Redis:", err)
		return err
	}
	err = r.redisClient.Set("chess_game:"+fmt.Sprint(game.ID), gameJSON, time.Minute*100) // Cache for 10 minutes
	if err != nil {
		log.Error("Error saving game to Redis:", err)
	}
	return err
}

func (r ChessRepositoryImpl) SaveGameStateToDB(game *dao.GameState) error {
	if err := r.db.Save(game).Error; err != nil {
		log.Error("Error saving game state to DB:", err)
		return err
	}
	return nil
}

func (r ChessRepositoryImpl) SaveGameMoveToDB(gameMove *dao.GameMove) error {
	if err := r.db.Save(gameMove).Error; err != nil {
		log.Error("Error saving game move to DB:", err)
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
