package locker

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	redis "github.com/redis/go-redis/v9"
)

func TestRedisLockOwnershipRenewalAndContention(t *testing.T) {
	addr := os.Getenv("SCHEDULETASK_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("set SCHEDULETASK_TEST_REDIS_ADDR to a disposable Redis instance")
	}
	client := redis.NewClient(&redis.Options{Addr: addr, Password: os.Getenv("SCHEDULETASK_TEST_REDIS_PASSWORD")})
	t.Cleanup(func() { client.Close() })
	lock := &rdLock{client: client}
	ctx := context.Background()
	key := fmt.Sprintf("test_lock_%d", time.Now().UnixNano())
	t.Cleanup(func() { client.Del(ctx, key) })
	if ok, err := lock.Lock(ctx, key, "owner", 5); err != nil || !ok {
		t.Fatalf("initial: %v, %v", ok, err)
	}
	if ok, err := lock.Lock(ctx, key, "other", 5); err != nil || ok {
		t.Fatalf("stolen: %v, %v", ok, err)
	}
	if err := lock.Unlock(ctx, key, "other"); err != nil {
		t.Fatal(err)
	}
	if ok, err := lock.Lock(ctx, key, "owner", 30); err != nil || !ok {
		t.Fatalf("renewal: %v, %v", ok, err)
	}
	ttl, err := client.TTL(ctx, key).Result()
	if err != nil || ttl < 25*time.Second {
		t.Fatalf("same owner did not renew TTL: %v, %v", ttl, err)
	}
	if err := lock.Expire(ctx, key, "other", 1); err != nil {
		t.Fatal(err)
	}
	ttl, _ = client.TTL(ctx, key).Result()
	if ttl < 25*time.Second {
		t.Fatalf("wrong owner shortened lease: %v", ttl)
	}
	if err := lock.Unlock(ctx, key, "owner"); err != nil {
		t.Fatal(err)
	}
	var winners atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ok, err := lock.Lock(ctx, key, fmt.Sprintf("contender-%d", i), 30)
			if err != nil {
				t.Error(err)
			}
			if ok {
				winners.Add(1)
			}
		}(i)
	}
	wg.Wait()
	if winners.Load() != 1 {
		t.Fatalf("expected one lock winner, got %d", winners.Load())
	}
}
