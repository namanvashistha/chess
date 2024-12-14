package pkg

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	log "github.com/sirupsen/logrus"
)

var ctx = context.Background()

type RedisClient struct {
	client *redis.Client
}

func NewRedisClient(addr, password string, db int) *RedisClient {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Test connection
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	return &RedisClient{client: rdb}
}

func (r *RedisClient) Set(key string, value interface{}, expiration time.Duration) error {
	err := r.client.Set(ctx, key, value, expiration).Err()
	if err != nil {
		log.Errorf("Failed to set key %s in Redis: %v", key, err)
		return err
	}
	return nil
}

func (r *RedisClient) Get(key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil // Key does not exist
		}
		log.Errorf("Failed to get key %s from Redis: %v", key, err)
		return "", err
	}
	return val, nil
}

func (r *RedisClient) Delete(key string) error {
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		log.Errorf("Failed to delete key %s from Redis: %v", key, err)
		return err
	}
	return nil
}
