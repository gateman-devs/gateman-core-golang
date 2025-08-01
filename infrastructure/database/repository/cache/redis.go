package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	redisClient "gateman.io/infrastructure/database/connection/cache"
	"gateman.io/infrastructure/logger"
)

var (
	redisRepo RedisRepository
)

type RedisRepository struct {
	Client *redis.Client
}

func (redisRepo *RedisRepository) preRequest() {
	if redisRepo.Client == nil {
		client, _ := redisClient.GetInstance()
		redisRepo.Client = client.Client
		logger.Info("redis repository initialisation complete")
	}
}

func (redisRepo *RedisRepository) CreateEntry(key string, payload interface{}, ttl time.Duration) bool {
	redisRepo.preRequest()
	ctx := context.Background()
	_, err := redisRepo.Client.Set(ctx, key, payload, ttl).Result()
	if err != nil {
		logger.Error("redis error occured while running CreateEntry", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		}, logger.LoggerOptions{
			Key:  "key",
			Data: key,
		})
		return false
	}

	logger.Info("redis CreateEntry completed")
	return true
}

func (redisRepo *RedisRepository) FindOne(key string) *string {
	redisRepo.preRequest()
	ctx := context.Background()

	result, err := redisRepo.Client.Get(ctx, key).Result()

	if err != nil {
		if err.Error() == "redis: nil" {
			return nil
		}
		logger.Error("redis error occured while running FindOne", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		}, logger.LoggerOptions{
			Key:  "key",
			Data: key,
		})
		return nil
	}

	logger.Info("redis FindOne completed")
	return &result
}

func (redisRepo *RedisRepository) FindOneByteArray(key string) *[]byte {
	redisRepo.preRequest()
	ctx := context.Background()

	result, err := redisRepo.Client.Get(ctx, key).Bytes()

	if err != nil {
		if err.Error() == "redis: nil" {
			return nil
		}
		logger.Error("redis error occured while running FindOneByteArray", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		}, logger.LoggerOptions{
			Key:  "key",
			Data: key,
		})
		return nil
	}

	logger.Info("redis FindOneByteArray completed")
	return &result
}

func (redisRepo *RedisRepository) DeleteOne(key string) bool {
	redisRepo.preRequest()
	ctx := context.Background()

	result, err := redisRepo.Client.Del(ctx, key).Result()

	if err != nil {
		logger.Error("redis error occured while running DeleteOne", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		}, logger.LoggerOptions{
			Key:  "key",
			Data: key,
		})
		return false
	}
	if int(result) != 1 {
		return false
	}

	logger.Info("redis DeleteOne completed")
	return true
}

func (redisRepo *RedisRepository) CreateInSortedSet(key string, score float64, member interface{}) int64 {
	redisRepo.preRequest()
	ctx := context.Background()
	added := redisRepo.Client.ZAdd(ctx, key, redis.Z{
		Score: score, Member: member,
	})

	if err := added.Err(); err != nil {
		logger.Error("redis error occured while running CreateInSortedSet", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		}, logger.LoggerOptions{
			Key:  "key",
			Data: key,
		}, logger.LoggerOptions{
			Key:  "socre",
			Data: score,
		}, logger.LoggerOptions{
			Key:  "member",
			Data: member,
		})
		return 0
	}

	logger.Info("redis CreateInSet completed")
	return added.Val()
}

func (redisRepo *RedisRepository) FindSortedSet(key string) *[]string {
	redisRepo.preRequest()
	ctx := context.Background()

	result := redisRepo.Client.ZRange(ctx, key, 0, -1)
	if err := result.Err(); err != nil {
		logger.Error("redis error occured while running FindSortedSet", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		}, logger.LoggerOptions{
			Key:  "key",
			Data: key,
		})
		return nil
	}
	if result == nil {
		return nil
	}

	logger.Info("redis FindSet completed")
	val := result.Val()
	return &val
}

func (redisRepo *RedisRepository) CreateInSet(key string, member interface{}, ttl time.Duration) int64 {
	redisRepo.preRequest()
	ctx := context.Background()
	added := redisRepo.Client.SAdd(ctx, key, member)

	if err := added.Err(); err != nil {
		logger.Error("redis error occured while running CreateInSet", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		}, logger.LoggerOptions{
			Key:  "key",
			Data: key,
		}, logger.LoggerOptions{
			Key:  "member",
			Data: member,
		})
		return 0
	}

	// Set TTL for the key if specified
	if ttl > 0 {
		redisRepo.Client.Expire(ctx, key, ttl)
	}

	logger.Info("redis CreateInSet completed")
	return added.Val()
}

func (redisRepo *RedisRepository) FindSet(key string) *[]string {
	redisRepo.preRequest()
	ctx := context.Background()

	result := redisRepo.Client.SMembers(ctx, key)
	if err := result.Err(); err != nil {
		logger.Error("redis error occured while running FindSet", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		}, logger.LoggerOptions{
			Key:  "key",
			Data: key,
		})
		return nil
	}
	if result == nil {
		return nil
	}

	logger.Info("redis FindSet completed")
	val := result.Val()
	return &val
}

func (redisRepo *RedisRepository) CountSetMembers(key string) *int64 {
	redisRepo.preRequest()
	ctx := context.Background()

	result := redisRepo.Client.SCard(ctx, key)
	if err := result.Err(); err != nil {
		logger.Error("redis error occured while running CountSetMembers", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		}, logger.LoggerOptions{
			Key:  "key",
			Data: key,
		})
		return nil
	}
	if result == nil {
		return nil
	}

	logger.Info("redis CountSetMembers completed")
	val := result.Val()
	return &val
}

func (redisRepo *RedisRepository) DoesItemExistInSet(key string, item string) bool {
	redisRepo.preRequest()
	ctx := context.Background()

	result := redisRepo.Client.SIsMember(ctx, key, item)
	if err := result.Err(); err != nil {
		logger.Error("redis error occured while running DoesItemExistInSet", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		}, logger.LoggerOptions{
			Key:  "key",
			Data: key,
		})
		return false
	}
	if result == nil {
		return false
	}
	logger.Info("redis DoesItemExistInSet completed")
	return result.Val()
}

func (redisRepo *RedisRepository) IncrementField(key string, amount int64) int64 {
	redisRepo.preRequest()
	ctx := context.Background()

	result := redisRepo.Client.IncrBy(ctx, key, amount)
	if err := result.Err(); err != nil {
		logger.Error("redis error occured while running IncrementField", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		}, logger.LoggerOptions{
			Key:  "key",
			Data: key,
		})
		return 0
	}
	if result == nil {
		return 0
	}
	logger.Info("redis IncrementField completed")
	return result.Val()
}
