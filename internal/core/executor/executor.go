package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Gopher0727/ScheduleTask/internal/core/model"
	"github.com/Gopher0727/ScheduleTask/internal/core/provider"
	"github.com/Gopher0727/ScheduleTask/internal/logger"
	"github.com/Gopher0727/ScheduleTask/internal/metrics"
	"github.com/Gopher0727/ScheduleTask/internal/utils"
)

type Executor struct {
	taskStore  provider.TaskStore
	timerStore provider.TimerStore
}

type NotifyHTTPParam struct {
	Method string            `json:"method"`
	URL    string            `json:"url"`
	Header map[string]string `json:"header"`
	Body   string            `json:"body"`
}

func New(taskStore provider.TaskStore, timerStore provider.TimerStore) *Executor {
	return &Executor{taskStore: taskStore, timerStore: timerStore}
}

func (e *Executor) Work(ctx context.Context, timerIDUnixKey string) {
	timerID, unix, err := utils.SplitTimerIDUnix(timerIDUnixKey)
	if err != nil {
		logger.L.Infof("SplitTimerIDUnix error: %v, key: %s", err, timerIDUnixKey)
		return
	}

	task, err := e.taskStore.GetByTimerIDAndRunTime(ctx, timerID, unix)
	if err != nil {
		logger.L.Infof("GetByTimerIDAndRunTime error: %v", err)
		return
	}
	if task == nil {
		logger.L.Infof("task not found, timerID: %d, unix: %d", timerID, unix)
		return
	}

	if task.Status != 0 {
		logger.L.Infof("task already executed, timerID: %d, unix: %d, status: %d", timerID, unix, task.Status)
		return
	}

	timer, err := e.timerStore.GetByID(ctx, timerID)
	if err != nil {
		logger.L.Infof("GetByID error: %v", err)
		return
	}
	if timer == nil {
		logger.L.Infof("timer not found, timerID: %d", timerID)
		return
	}

	if timer.Status != 1 {
		logger.L.Infof("timer not enabled, timerID: %d, status: %d", timerID, timer.Status)
		return
	}

	claimed, err := e.taskStore.Claim(ctx, task.TaskID)
	if err != nil || !claimed {
		return
	}
	e.executeAndPostProcess(ctx, task, timer)
}

func (e *Executor) executeAndPostProcess(ctx context.Context, task *model.Task, timer *model.Timer) {
	start := time.Now()

	defer func() {
		metrics.TaskDuration.Observe(time.Since(start).Seconds())
	}()

	gapTime := int(time.Now().UnixMilli() - task.RunTimer)
	task.CostTime = gapTime

	resp, err := e.executeTimerCallback(ctx, timer)
	if err != nil {
		logger.L.Infof("executeTimerCallback error: %v", err)
		task.Status = 2
		task.Output = err.Error()
	} else if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		task.Status = 1
		task.Output = resp.Body
	} else {
		task.Status = 2
		task.Output = resp.Body
	}

	// Persist a terminal cancellation result after the worker context has ended.
	cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cleanupCancel()
	if err := e.taskStore.Update(cleanupCtx, task); err != nil {
		logger.L.Infof("Update task error: %v", err)
	}

	switch task.Status {
	case 1:
		metrics.TaskExecuted.WithLabelValues("success").Inc()
	case 2:
		metrics.TaskExecuted.WithLabelValues("failed").Inc()
	}
}

func (e *Executor) executeTimerCallback(ctx context.Context, timer *model.Timer) (*CallbackResponse, error) {
	var param NotifyHTTPParam
	if err := json.Unmarshal([]byte(timer.NotifyHTTPParam), &param); err != nil {
		return nil, fmt.Errorf("unmarshal NotifyHTTPParam error: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}

	var req *http.Request
	var err error

	switch strings.ToUpper(param.Method) {
	case "POST":
		req, err = http.NewRequestWithContext(ctx, "POST", param.URL, strings.NewReader(param.Body))
	default:
		return nil, fmt.Errorf("unsupported method: %s", param.Method)
	}

	if err != nil {
		return nil, fmt.Errorf("create request error: %w", err)
	}

	for k, v := range param.Header {
		req.Header.Set(k, v)
	}
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if err != nil {
		return nil, fmt.Errorf("read response error: %w", err)
	}
	if len(body) > 1<<20 {
		return nil, fmt.Errorf("callback response exceeds 1 MiB")
	}

	return &CallbackResponse{
		StatusCode: resp.StatusCode,
		Body:       string(body),
	}, nil
}

type CallbackResponse struct {
	StatusCode int
	Body       string
}
