package redis

import (
	"time"

	redis "github.com/redis/go-redis/v9"

	"github.com/Gopher0727/ScheduleTask/internal/config/infra"
)

func New(cfg infra.Redis) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: cfg.Addr, Password: cfg.Password, DB: cfg.DB, PoolSize: cfg.PoolSize,
		DialTimeout: 3 * time.Second, ReadTimeout: 3 * time.Second, WriteTimeout: 3 * time.Second,
	})
}
