package timer

import (
	"context"
	"testing"

	"github.com/Gopher0727/ScheduleTask/internal/core/model"
	"github.com/Gopher0727/ScheduleTask/internal/testutil"
)

func TestStoreAssignsIDsAndUpdatesExistingTimer(t *testing.T) {
	db := testutil.Postgres(t)
	if err := db.AutoMigrate(&model.Timer{}); err != nil {
		t.Fatal(err)
	}
	store := &Store{DB: db}
	ctx := context.Background()
	first := &model.Timer{App: "demo", Name: "first", Cron: "* * * * *"}
	second := &model.Timer{App: "demo", Name: "second", Cron: "* * * * *"}
	if err := store.Save(ctx, first); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, second); err != nil {
		t.Fatal(err)
	}
	if first.TimerID <= 0 || second.TimerID == first.TimerID {
		t.Fatalf("invalid generated IDs: %d, %d", first.TimerID, second.TimerID)
	}
	if first.CreateTime.IsZero() || first.UpdateTime.IsZero() {
		t.Fatal("timestamps not populated")
	}
	first.Status = 1
	if err := store.Update(ctx, first); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetByID(ctx, first.TimerID)
	if err != nil || got == nil || got.Status != 1 {
		t.Fatalf("update not persisted: %+v, %v", got, err)
	}
	enabled, err := store.GetByStatus(ctx, 1)
	if err != nil || len(enabled) != 1 {
		t.Fatalf("enabled: %+v, %v", enabled, err)
	}
	missing, err := store.GetByID(ctx, 99999)
	if err != nil || missing != nil {
		t.Fatalf("missing: %+v, %v", missing, err)
	}
}
