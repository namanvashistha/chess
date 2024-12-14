package config

import "chess-engine/app/pkg"

func InitRedis() *pkg.RedisClient {
	// Replace with your Redis configuration
	const redisAddr = "localhost:6379"
	const redisPassword = "" // Leave empty if no password
	const redisDB = 0        // Default DB

	return pkg.NewRedisClient(redisAddr, redisPassword, redisDB)
}
