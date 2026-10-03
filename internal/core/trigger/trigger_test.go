package trigger

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Gopher0727/ScheduleTask/internal/config"
	"github.com/Gopher0727/ScheduleTask/internal/core/model"
	"github.com/Gopher0727/ScheduleTask/internal/logger"
)

type wheelStub struct {
	err     error
	members []string
}

func (w wheelStub) Add(context.Context, string, string, float64) error { return nil }
func (w wheelStub) Remove(context.Context, string, ...string) error    { return nil }
func (w wheelStub) RangeByScore(context.Context, string, float64, float64) ([]string, error) {
	return w.members, w.err
}

type tasksStub struct{ items []*model.Task }

func (s tasksStub) Claim(context.Context, int64) (bool, error) { return true, nil }
func (s tasksStub) ListByTimerID(context.Context, int64, int) ([]*model.Task, error) {
	return s.items, nil
}
func (s tasksStub) BatchSave(context.Context, []*model.Task) error { return nil }
func (s tasksStub) Update(context.Context, *model.Task) error      { return nil }
func (s tasksStub) GetByTimerIDAndRunTime(context.Context, int64, int64) (*model.Task, error) {
	return nil, nil
}
func (s tasksStub) GetByTimeRange(context.Context, int64, int64, int) ([]*model.Task, error) {
	return s.items, nil
}

type execStub struct{ calls chan string }

func (e execStub) Work(_ context.Context, key string) { e.calls <- key }

func TestBucketTimeUsesLocalZone(t *testing.T) {
	old := time.Local
	time.Local = time.FixedZone("CST", 8*60*60)
	defer func() { time.Local = old }()
	got, err := parseStartTime("2026-10-03 12:00_4")
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 10, 3, 4, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("bucket parsed as %v, want %v", got, want)
	}
}
func TestFallbackOnlyReturnsCurrentBucket(t *testing.T) {
	logger.Init("info")
	cfg, err := config.Load("../../../config.example.toml")
	if err != nil {
		t.Fatal(err)
	}
	tr := New(wheelStub{err: errors.New("unavailable")}, tasksStub{items: []*model.Task{{TimerID: 1}, {TimerID: 2}, {TimerID: 6}}}, nil, cfg.Trigger, cfg.Scheduler.BucketNum)
	got, err := tr.getTasksByTime(context.Background(), time.Now(), time.Now(), "2026-10-03 12:00_1")
	if err != nil || len(got) != 2 || got[0].TimerID != 1 || got[1].TimerID != 6 {
		t.Fatalf("fallback crossed buckets: %+v, %v", got, err)
	}
}
func TestPastBucketIsScannedImmediately(t *testing.T) {
	cfg, err := config.Load("../../../config.example.toml")
	if err != nil {
		t.Fatal(err)
	}
	calls := make(chan string, 1)
	tr := New(wheelStub{members: []string{"7_1000"}}, tasksStub{}, execStub{calls}, cfg.Trigger, cfg.Scheduler.BucketNum)
	key := time.Now().Add(-time.Minute).Format("2006-01-02 15:04") + "_2"
	done := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { tr.Work(ctx, key); close(done) }()
	select {
	case got := <-calls:
		if got != "7_1000" {
			t.Fatal(got)
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatal("past due bucket waited for ticker")
	}
	select {
	case <-done:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("finished bucket kept running")
	}
}

func TestMissingOrPartialRedisIndexRecoversPendingTasks(t *testing.T) {
	cfg, err := config.Load("../../../config.example.toml")
	if err != nil {
		t.Fatal(err)
	}
	for _, members := range [][]string{nil, {"1_1000"}} {
		tr := New(wheelStub{members: members}, tasksStub{items: []*model.Task{{TimerID: 1, RunTimer: 1000}, {TimerID: 6, RunTimer: 2000}, {TimerID: 2, RunTimer: 1000}}}, nil, cfg.Trigger, cfg.Scheduler.BucketNum)
		got, err := tr.getTasksByTime(context.Background(), time.UnixMilli(0), time.UnixMilli(3000), "2026-10-03 12:00_1")
		if err != nil || len(got) != 2 {
			t.Fatalf("lost pending executions with index %v: %+v, %v", members, got, err)
		}
		seen := map[int64]bool{}
		for _, task := range got {
			seen[task.TimerID] = true
		}
		if !seen[1] || !seen[6] || seen[2] {
			t.Fatalf("recovery crossed buckets: %+v", got)
		}
	}
}
