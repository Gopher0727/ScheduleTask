package task

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Gopher0727/ScheduleTask/internal/core/model"
	"github.com/Gopher0727/ScheduleTask/internal/testutil"
)

func TestBatchSaveIsIdempotentAndPreservesResult(t *testing.T) {
	db := testutil.Postgres(t)
	if err := db.AutoMigrate(&model.Task{}); err != nil {
		t.Fatal(err)
	}
	store := &Store{DB: db}
	ctx := context.Background()
	item := &model.Task{TimerID: 7, RunTimer: 1000, App: "demo"}
	if err := store.BatchSave(ctx, []*model.Task{item}); err != nil {
		t.Fatal(err)
	}
	if item.TaskID <= 0 {
		t.Fatalf("missing task ID: %d", item.TaskID)
	}
	item.Status = 1
	item.Output = "delivered"
	if err := store.Update(ctx, item); err != nil {
		t.Fatal(err)
	}
	if err := store.BatchSave(ctx, []*model.Task{{TimerID: 7, RunTimer: 1000, App: "demo"}}); err != nil {
		t.Fatal(err)
	}
	var count int64
	db.Model(&model.Task{}).Count(&count)
	if count != 1 {
		t.Fatalf("duplicate materialization created %d tasks", count)
	}
	got, err := store.GetByTimerIDAndRunTime(ctx, 7, 1000)
	if err != nil || got.Status != 1 || got.Output != "delivered" {
		t.Fatalf("result lost: %+v, %v", got, err)
	}
	if err := store.BatchSave(ctx, nil); err != nil {
		t.Fatal(err)
	}
}

func TestClaimHasOneOwnerAndHistoryIsBounded(t *testing.T) {
	db := testutil.Postgres(t)
	if err := db.AutoMigrate(&model.Task{}); err != nil {
		t.Fatal(err)
	}
	store := &Store{DB: db}
	ctx := context.Background()
	item := &model.Task{TimerID: 7, RunTimer: 1000, App: "demo"}
	if err := store.BatchSave(ctx, []*model.Task{item, {TimerID: 7, RunTimer: 2000, App: "demo"}}); err != nil {
		t.Fatal(err)
	}
	var winners atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := store.Claim(ctx, item.TaskID)
			if err != nil {
				t.Error(err)
			}
			if ok {
				winners.Add(1)
			}
		}()
	}
	wg.Wait()
	if winners.Load() != 1 {
		t.Fatalf("expected one execution owner, got %d", winners.Load())
	}
	history, err := store.ListByTimerID(ctx, 7, 1)
	if err != nil || len(history) != 1 || history[0].RunTimer != 2000 {
		t.Fatalf("history: %+v, %v", history, err)
	}
	got, err := store.GetByTimerIDAndRunTime(ctx, 7, 1000)
	if err != nil || got.Status != 3 {
		t.Fatalf("claim not persisted: %+v, %v", got, err)
	}
}
