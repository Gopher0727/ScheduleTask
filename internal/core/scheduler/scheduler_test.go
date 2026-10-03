package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/Gopher0727/ScheduleTask/internal/config"
)

type alwaysLock struct{}

func (alwaysLock) Lock(context.Context, string, string, int64) (bool, error) { return true, nil }
func (alwaysLock) Unlock(context.Context, string, string) error              { return nil }
func (alwaysLock) Expire(context.Context, string, string, int64) error       { return nil }

type blockedTrigger struct{ started chan string }

func (t blockedTrigger) Work(ctx context.Context, key string) { t.started <- key; <-ctx.Done() }

func TestBucketsStartConcurrentlyAndStopOnCancellation(t *testing.T) {
	cfg, err := config.Load("../../../config.example.toml")
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan string, 10)
	s := New(alwaysLock{}, blockedTrigger{started}, cfg.Scheduler)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { s.Run(ctx); close(done) }()
	for i := 0; i < 10; i++ {
		select {
		case <-started:
		case <-time.After(3 * time.Second):
			t.Fatalf("only %d buckets started; one blocked the others", i)
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("scheduler did not join workers")
	}
}
