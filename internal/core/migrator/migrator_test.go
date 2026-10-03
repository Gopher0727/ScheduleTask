package migrator

import (
	"context"
	"testing"
	"time"

	"github.com/Gopher0727/ScheduleTask/internal/config"
	"github.com/Gopher0727/ScheduleTask/internal/core/model"
)

func TestExecutionTimesKeepIntervalAnchor(t *testing.T) {
	origin := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	timer := &model.Timer{Cron: "@every 10m", CreateTime: origin}
	for _, now := range []time.Time{origin, origin.Add(time.Minute)} {
		times, err := executionTimes(context.Background(), timer, now, origin.Add(30*time.Minute))
		if err != nil || len(times) != 2 || times[0] != origin.Add(10*time.Minute).UnixMilli() || times[1] != origin.Add(20*time.Minute).UnixMilli() {
			t.Fatalf("interval drift at %v: %v, %v", now, times, err)
		}
	}
}

func TestExecutionTimesTerminateForImpossibleDateAndCancellation(t *testing.T) {
	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	times, err := executionTimes(ctx, &model.Timer{Cron: "0 0 30 2 *"}, now, now.Add(time.Hour))
	if err != nil || len(times) != 0 {
		t.Fatalf("impossible date: %v, %v", times, err)
	}
	cancel()
	if _, err := executionTimes(ctx, &model.Timer{Cron: "* * * * *"}, now, now.Add(time.Hour)); err == nil {
		t.Fatal("canceled materialization succeeded")
	}
}

type timerStub struct{ scanned chan struct{} }

func (s timerStub) Save(context.Context, *model.Timer) error             { return nil }
func (s timerStub) Update(context.Context, *model.Timer) error           { return nil }
func (s timerStub) GetByID(context.Context, int64) (*model.Timer, error) { return nil, nil }
func (s timerStub) GetByStatus(context.Context, int) ([]*model.Timer, error) {
	s.scanned <- struct{}{}
	return nil, nil
}

type lockStub struct{}

func (lockStub) Lock(context.Context, string, string, int64) (bool, error) { return true, nil }
func (lockStub) Unlock(context.Context, string, string) error              { return nil }
func (lockStub) Expire(context.Context, string, string, int64) error       { return nil }
func TestMigrationRunsAtStartup(t *testing.T) {
	cfg, err := config.Load("../../../config.example.toml")
	if err != nil {
		t.Fatal(err)
	}
	scanned := make(chan struct{}, 1)
	m := New(timerStub{scanned}, nil, nil, lockStub{}, cfg.Migrator, cfg.Scheduler.BucketNum)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { m.Run(ctx); close(done) }()
	select {
	case <-scanned:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("no startup migration")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("migration did not stop")
	}
}
