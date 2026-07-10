package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gamatritunggal/smartscan/backend/internal/config"
	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func ConnectRedis(cfg *config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 2,
		PoolTimeout:  4 * time.Second,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close() // Close client on connection failure
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	RedisClient = client
	log.Println("Redis connected successfully")
	return client, nil
}

func CloseRedis() error {
	if RedisClient != nil {
		return RedisClient.Close()
	}
	return nil
}
