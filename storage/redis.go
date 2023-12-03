package storage

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"cncamp/pkg/third_party/nightingale/pkg/tlsx"
	"github.com/redis/go-redis/v9"
)

type RedisConfig struct {
	Address  string
	Username string
	DataDir  string
	Password string
	DB       int
	UseTLS   bool
	tlsx.ClientConfig
	RedisType        string
	MasterName       string
	SentinelUsername string
	SentinelPassword string
}

type Redis interface {
	// Pipeline() redis.Pipeliner
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	MSet(ctx context.Context, values ...interface{}) *redis.StatusCmd
	MGet(ctx context.Context, keys ...string) *redis.SliceCmd
	// HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd
	// HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	// HDel(ctx context.Context, key string, fields ...string) *redis.IntCmd
	Close() error
	Ping(ctx context.Context) *redis.StatusCmd
	// Publish(ctx context.Context, channel string, message interface{}) *redis.IntCmd
}

func NewRedis(cfg RedisConfig) (Redis, error) {
	var redisClient Redis
	switch cfg.RedisType {
	case "standalone", "":
		redisOptions := &redis.Options{
			Addr:     cfg.Address,
			Username: cfg.Username,
			Password: cfg.Password,
			DB:       cfg.DB,
		}

		if cfg.UseTLS {
			tlsConfig, err := cfg.TLSConfig()
			if err != nil {
				fmt.Println("failed to init redis tls config:", err)
				os.Exit(1)
			}
			redisOptions.TLSConfig = tlsConfig
		}

		redisClient = redis.NewClient(redisOptions)

	case "cluster":
		redisOptions := &redis.ClusterOptions{
			Addrs:    strings.Split(cfg.Address, ","),
			Username: cfg.Username,
			Password: cfg.Password,
		}

		if cfg.UseTLS {
			tlsConfig, err := cfg.TLSConfig()
			if err != nil {
				fmt.Println("failed to init redis tls config:", err)
				os.Exit(1)
			}
			redisOptions.TLSConfig = tlsConfig
		}

		redisClient = redis.NewClusterClient(redisOptions)

	case "sentinel":
		redisOptions := &redis.FailoverOptions{
			MasterName:       cfg.MasterName,
			SentinelAddrs:    strings.Split(cfg.Address, ","),
			Username:         cfg.Username,
			Password:         cfg.Password,
			DB:               cfg.DB,
			SentinelUsername: cfg.SentinelUsername,
			SentinelPassword: cfg.SentinelPassword,
		}

		if cfg.UseTLS {
			tlsConfig, err := cfg.TLSConfig()
			if err != nil {
				fmt.Println("failed to init redis tls config:", err)
				os.Exit(1)
			}
			redisOptions.TLSConfig = tlsConfig
		}

		redisClient = redis.NewFailoverClient(redisOptions)

	default:
		fmt.Println("failed to init redis , redis type is illegal:", cfg.RedisType)
		os.Exit(1)
	}

	err := redisClient.Ping(context.Background()).Err()
	if err != nil {
		fmt.Println("failed to ping redis:", err)
		os.Exit(1)
	}
	return redisClient, nil
}

// func MGet(ctx context.Context, r Redis, keys []string) ([][]byte, error) {
// 	var vals [][]byte
// 	pipe := r.Pipeline()
// 	for _, key := range keys {
// 		pipe.Get(ctx, key)
// 	}
// 	cmds, err := pipe.Exec(ctx)
// 	if err != nil {
// 		logger.Errorf("failed to exec pipeline: %s", err)
// 		return vals, err
// 	}

// 	for i, key := range keys {
// 		cmd := cmds[i]
// 		if cmd.Err() != nil {
// 			logger.Errorf("failed to get key: %s, err: %s", key, cmd.Err())
// 			continue
// 		}
// 		val := []byte(cmd.(*redis.StringCmd).Val())
// 		vals = append(vals, val)
// 	}

// 	return vals, err
// }

// func MSet(ctx context.Context, r Redis, m map[string]interface{}) error {
// 	pipe := r.Pipeline()
// 	for k, v := range m {
// 		pipe.Set(ctx, k, v, 0)
// 	}
// 	_, err := pipe.Exec(ctx)
// 	return err
// }
