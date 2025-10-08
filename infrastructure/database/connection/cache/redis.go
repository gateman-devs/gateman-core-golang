package cache

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	Client *redis.Client
	ctx    context.Context
}

var (
	redisInstance *RedisClient
	redisOnce     sync.Once
)

type Config struct {
	// Connection URL (takes precedence if provided)
	URL string

	// Individual connection parameters (used if URL is not provided)
	Addr            string
	Password        string
	DB              int
	PoolSize        int
	MinIdleConns    int
	MaxIdleConns    int
	ConnMaxIdleTime time.Duration
	ConnMaxLifetime time.Duration
	DialTimeout     time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
}

func GetDefaultConfig() *Config {
	return &Config{
		URL: os.Getenv("REDIS_URL"),
		DB:              0,                // Default Redis DB
		PoolSize:        100,              // Maximum number of socket connections
		MinIdleConns:    10,               // Minimum number of idle connections
		MaxIdleConns:    30,               // Maximum number of idle connections
		ConnMaxIdleTime: 30 * time.Minute, // Maximum amount of time a connection may be idle
		ConnMaxLifetime: time.Hour,        // Maximum amount of time a connection may be reused
		DialTimeout:     5 * time.Second,  // Timeout for establishing new connections
		ReadTimeout:     3 * time.Second,  // Timeout for socket reads
		WriteTimeout:    3 * time.Second,  // Timeout for socket writes
	}
}

func Connect(config *Config) *RedisClient {
	if config == nil {
		config = GetDefaultConfig()
	}

	var rdb *redis.Client

	// If URL is provided, use it (takes precedence)
	if config.URL != "" {
		opt, err := redis.ParseURL(config.URL)
		if err != nil {
			panic(fmt.Errorf("failed to parse Redis URL: %w", err))
		}

		// Override with custom configuration if provided
		if config.PoolSize > 0 {
			opt.PoolSize = config.PoolSize
		}
		if config.MinIdleConns > 0 {
			opt.MinIdleConns = config.MinIdleConns
		}
		if config.MaxIdleConns > 0 {
			opt.MaxIdleConns = config.MaxIdleConns
		}
		if config.ConnMaxIdleTime > 0 {
			opt.ConnMaxIdleTime = config.ConnMaxIdleTime
		}
		if config.ConnMaxLifetime > 0 {
			opt.ConnMaxLifetime = config.ConnMaxLifetime
		}
		if config.DialTimeout > 0 {
			opt.DialTimeout = config.DialTimeout
		}
		if config.ReadTimeout > 0 {
			opt.ReadTimeout = config.ReadTimeout
		}
		if config.WriteTimeout > 0 {
			opt.WriteTimeout = config.WriteTimeout
		}

		rdb = redis.NewClient(opt)
	} else {
		// Fallback to individual parameters
		rdb = redis.NewClient(&redis.Options{
			Addr:            config.Addr,
			Password:        config.Password,
			DB:              config.DB,
			PoolSize:        config.PoolSize,
			MinIdleConns:    config.MinIdleConns,
			MaxIdleConns:    config.MaxIdleConns,
			ConnMaxIdleTime: config.ConnMaxIdleTime,
			ConnMaxLifetime: config.ConnMaxLifetime,
			DialTimeout:     config.DialTimeout,
			ReadTimeout:     config.ReadTimeout,
			WriteTimeout:    config.WriteTimeout,
		})
	}

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		panic(fmt.Errorf("failed to ping Redis: %w", err))
	}

	redisClient := &RedisClient{
		Client: rdb,
		ctx:    ctx,
	}

	log.Printf("Successfully connected to Redis")
	return redisClient
}

func GetInstance() (*RedisClient, error) {
	var err error
	redisOnce.Do(func() {
		redisInstance = Connect(nil)
	})
	return redisInstance, err
}

func (rc *RedisClient) Close() error {
	if rc.Client != nil {
		if err := rc.Client.Close(); err != nil {
			return fmt.Errorf("failed to close Redis connection: %w", err)
		}
		log.Println("Successfully closed Redis connection")
	}
	return nil
}
