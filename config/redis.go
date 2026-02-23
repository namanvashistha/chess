package config

import (
	"chess-engine/app/pkg"
	"os"
)

func InitRedis() *pkg.RedisClient {
	// Replace with your Redis configuration
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	const redisPassword = "" // Leave empty if no password
	const redisDB = 0        // Default DB

	return pkg.NewRedisClient(redisAddr, redisPassword, redisDB)
}
