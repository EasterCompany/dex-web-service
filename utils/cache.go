package utils

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var RDB *redis.Client

// DEPRECATED: Initialize RDB via network.NewBrain("web").Stem().Client() in main.go
func GetRedisClient() *redis.Client {
	return RDB
}

func GetWebViewCache(ctx context.Context, targetURL string) (string, error) {
	if RDB == nil {
		return "", fmt.Errorf("redis not initialized")
	}

	key := fmt.Sprintf("web:cache:%x", sha256.Sum256([]byte(targetURL)))
	return RDB.Get(ctx, key).Result()
}

func SetWebViewCache(ctx context.Context, targetURL string, content string) error {
	if RDB == nil {
		return fmt.Errorf("redis not initialized")
	}

	key := fmt.Sprintf("web:cache:%x", sha256.Sum256([]byte(targetURL)))
	return RDB.Set(ctx, key, content, 10*time.Minute).Err()
}
