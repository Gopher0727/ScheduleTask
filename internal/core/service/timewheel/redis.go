package timewheel

import (
	"context"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"
)

type rdTW struct {
	client *redis.Client
}

func (w *rdTW) Add(ctx context.Context, key, member string, score float64) error {
	// Buckets are disposable indexes; PostgreSQL remains the source of truth.
	ttl := time.Until(time.UnixMilli(int64(score))) + 2*time.Minute
	if ttl < 2*time.Minute {
		ttl = 2 * time.Minute
	}
	_, err := w.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.ZAdd(ctx, key, redis.Z{Score: score, Member: member})
		pipe.Expire(ctx, key, ttl)
		return nil
	})
	return err
}

func (w *rdTW) RangeByScore(ctx context.Context, key string, min, max float64) ([]string, error) {
	return w.client.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: fmt.Sprintf("%f", min), Max: fmt.Sprintf("%f", max),
	}).Result()
}

func (w *rdTW) Remove(ctx context.Context, key string, members ...string) error {
	if len(members) == 0 {
		return nil
	}
	return w.client.ZRem(ctx, key, members).Err()
}

func New(client *redis.Client) *rdTW { return &rdTW{client: client} }
