package executor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Gopher0727/ScheduleTask/internal/core/model"
	taskStore "github.com/Gopher0727/ScheduleTask/internal/core/service/task"
	"github.com/Gopher0727/ScheduleTask/internal/logger"
	"github.com/Gopher0727/ScheduleTask/internal/testutil"
)

type taskStub struct{}

func (taskStub) Claim(context.Context, int64) (bool, error)                       { return true, nil }
func (taskStub) ListByTimerID(context.Context, int64, int) ([]*model.Task, error) { return nil, nil }
func (taskStub) BatchSave(context.Context, []*model.Task) error                   { return nil }
func (taskStub) Update(context.Context, *model.Task) error                        { return nil }
func (taskStub) GetByTimerIDAndRunTime(context.Context, int64, int64) (*model.Task, error) {
	return nil, nil
}

func TestCallbackRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(strings.Repeat("x", (1<<20)+1))) }))
	defer server.Close()
	param, _ := json.Marshal(NotifyHTTPParam{Method: "POST", URL: server.URL})
	if _, err := New(taskStub{}, nil).executeTimerCallback(context.Background(), &model.Timer{NotifyHTTPParam: string(param)}); err == nil {
		t.Fatal("oversized callback response was accepted")
	}
}

func TestCanceledCallbackPersistsTerminalStatus(t *testing.T) {
	db := testutil.Postgres(t)
	if err := db.AutoMigrate(&model.Task{}); err != nil {
		t.Fatal(err)
	}
	store := taskStore.New(db)
	item := &model.Task{TimerID: 1, RunTimer: 1000}
	if err := store.BatchSave(context.Background(), []*model.Task{item}); err != nil {
		t.Fatal(err)
	}
	if ok, err := store.Claim(context.Background(), item.TaskID); err != nil || !ok {
		t.Fatalf("claim: %v, %v", ok, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	param, _ := json.Marshal(NotifyHTTPParam{Method: "POST", URL: "http://127.0.0.1:1/callback"})
	New(store, nil).executeAndPostProcess(ctx, item, &model.Timer{NotifyHTTPParam: string(param)})
	got, err := store.GetByTimerIDAndRunTime(context.Background(), 1, 1000)
	if err != nil || got.Status != 2 {
		t.Fatalf("canceled callback remained running: %+v, %v", got, err)
	}
}
func (taskStub) GetByTimeRange(context.Context, int64, int64, int) ([]*model.Task, error) {
	return nil, nil
}
func TestCanceledExecutionDoesNotSendCallback(t *testing.T) {
	logger.Init("info")
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(200) }))
	defer server.Close()
	param, _ := json.Marshal(NotifyHTTPParam{Method: "POST", URL: server.URL})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	e := New(taskStub{}, nil)
	e.executeAndPostProcess(ctx, &model.Task{}, &model.Timer{NotifyHTTPParam: string(param)})
	if calls.Load() != 0 {
		t.Fatalf("canceled work sent %d callbacks", calls.Load())
	}
}
