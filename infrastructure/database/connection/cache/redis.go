package cache

import (
	"os"

	"authone.usepolymer.co/infrastructure/logger"
	"github.com/go-redis/redis"
)

var (
	Client *redis.Client
)

func connectRedis() {
	opt := &redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
		PoolSize: 10,
	}
	Client = redis.NewClient(opt)
	logger.Info("connected to redis successfully")
}
