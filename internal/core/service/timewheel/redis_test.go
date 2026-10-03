package timewheel

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	redis "github.com/redis/go-redis/v9"
)

func TestRedisTimeWheelUpsertRangeAndExpiry(t *testing.T) {
	addr := os.Getenv("SCHEDULETASK_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("set SCHEDULETASK_TEST_REDIS_ADDR to a disposable Redis instance")
	}
	client := redis.NewClient(&redis.Options{Addr: addr, Password: os.Getenv("SCHEDULETASK_TEST_REDIS_PASSWORD")})
	t.Cleanup(func() { client.Close() })
	wheel := &rdTW{client: client}
	ctx := context.Background()
	key := fmt.Sprintf("test_wheel_%d", time.Now().UnixNano())
	t.Cleanup(func() { client.Del(ctx, key) })
	now := time.Now().UnixMilli()
	if err := wheel.Add(ctx, key, "7", float64(now)); err != nil {
		t.Fatal(err)
	}
	if err := wheel.Add(ctx, key, "7", float64(now+1000)); err != nil {
		t.Fatal(err)
	}
	if err := wheel.Add(ctx, key, "8", float64(now+2000)); err != nil {
		t.Fatal(err)
	}
	got, err := wheel.RangeByScore(ctx, key, float64(now), float64(now+1999))
	if err != nil || len(got) != 1 || got[0] != "7" {
		t.Fatalf("range: %v, %v", got, err)
	}
	ttl, err := client.TTL(ctx, key).Result()
	if err != nil || ttl <= 0 {
		t.Fatalf("bucket has no TTL: %v, %v", ttl, err)
	}
	if err := wheel.Remove(ctx, key, "7"); err != nil {
		t.Fatal(err)
	}
	got, err = wheel.RangeByScore(ctx, key, 0, float64(now+3000))
	if err != nil || len(got) != 1 || got[0] != "8" {
		t.Fatalf("remove: %v, %v", got, err)
	}
}
