package utils

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var RDB *redis.Client

// InitRedis initializes the Redis client using the local-cache-0 definition
func InitRedis(addr string, password string, db int) {
	RDB = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := RDB.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Failed to connect to Redis for caching: %v", err)
	} else {
		log.Printf("Connected to Redis for web caching at %s", addr)
	}
}

// GetRedisClient returns the global Redis client
func GetRedisClient() *redis.Client {
	return RDB
}

// GetCachedPage returns the raw HTML of a page if it exists in cache
func GetCachedPage(ctx context.Context, targetURL string) (string, error) {
	if RDB == nil {
		return "", fmt.Errorf("redis not initialized")
	}

	key := fmt.Sprintf("web:cache:%x", sha256.Sum256([]byte(targetURL)))
	return RDB.Get(ctx, key).Result()
}

// SetCachedPage stores the raw HTML of a page in cache with a 10-minute TTL
func SetCachedPage(ctx context.Context, targetURL string, content string) error {
	if RDB == nil {
		return fmt.Errorf("redis not initialized")
	}

	key := fmt.Sprintf("web:cache:%x", sha256.Sum256([]byte(targetURL)))
	return RDB.Set(ctx, key, content, 10*time.Minute).Err()
}
