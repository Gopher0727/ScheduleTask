package trigger

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Gopher0727/ScheduleTask/internal/config/core"
	"github.com/Gopher0727/ScheduleTask/internal/core/model"
	"github.com/Gopher0727/ScheduleTask/internal/core/provider"
	"github.com/Gopher0727/ScheduleTask/internal/logger"
)

type Trigger struct {
	timeWheel provider.TimeWheel
	taskCache provider.TaskStore
	executor  Executor
	cfg       core.Trigger
	bucketNum int
}

type Executor interface {
	Work(ctx context.Context, timerIDUnixKey string)
}

func New(timeWheel provider.TimeWheel, taskCache provider.TaskStore, executor Executor, cfg core.Trigger, bucketNum int) *Trigger {
	return &Trigger{timeWheel: timeWheel, taskCache: taskCache, executor: executor, cfg: cfg, bucketNum: bucketNum}
}

func (t *Trigger) Work(ctx context.Context, minuteBucketKey string) {
	startTime, err := parseStartTime(minuteBucketKey)
	if err != nil {
		logger.L.Infof("parseStartTime error: %v", err)
		return
	}
	endTime := startTime.Add(1 * time.Minute)

	ticker := time.NewTicker(time.Duration(t.cfg.ZRangeGapSeconds) * time.Second)
	defer ticker.Stop()

	cursor := startTime.UnixMilli()
	scan := func() bool {
		now := time.Now()
		last := now.UnixMilli()
		if last >= endTime.UnixMilli() {
			last = endTime.UnixMilli() - 1
		}
		if last >= cursor && t.handleBatch(ctx, time.UnixMilli(cursor), time.UnixMilli(last+1), minuteBucketKey) {
			cursor = last + 1
		}
		return !now.Before(endTime) && cursor >= endTime.UnixMilli()
	}
	if ctx.Err() != nil || scan() {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if scan() {
				return
			}
		}
	}
}

func (t *Trigger) handleBatch(ctx context.Context, start, end time.Time, minuteBucketKey string) bool {
	tasks, err := t.getTasksByTime(ctx, start, end, minuteBucketKey)
	if err != nil {
		logger.L.Infof("getTasksByTime error: %v", err)
		return false
	}

	for _, task := range tasks {
		if ctx.Err() != nil {
			return false
		}
		timerIDUnixKey := fmt.Sprintf("%d_%d", task.TimerID, task.RunTimer)
		t.executor.Work(ctx, timerIDUnixKey)
	}
	return true
}

func (t *Trigger) getTasksByTime(ctx context.Context, start, end time.Time, minuteBucketKey string) ([]*model.Task, error) {
	members, err := t.timeWheel.RangeByScore(ctx, minuteBucketKey, float64(start.UnixMilli()), float64(end.UnixMilli()-1))
	if err != nil {
		logger.L.Infof("RangeByScore error, reconciling with taskStore: %v", err)
	}
	// Reconcile even a healthy partial/empty Redis index against durable pending tasks.
	pending, err := t.taskCache.GetByTimeRange(ctx, start.UnixMilli(), end.UnixMilli()-1, 0)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(minuteBucketKey, "_")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid minuteBucketKey: %s", minuteBucketKey)
	}
	bucket, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool, len(members))
	for _, member := range members {
		seen[member] = true
	}
	for _, task := range pending {
		if task.TimerID%int64(t.bucketNum) != bucket {
			continue
		}
		member := fmt.Sprintf("%d_%d", task.TimerID, task.RunTimer)
		if !seen[member] {
			members = append(members, member)
			seen[member] = true
		}
	}

	var tasks []*model.Task
	for _, member := range members {
		timerID, unix, err := parseMember(member)
		if err != nil {
			logger.L.Infof("parseMember error: %v, member: %s", err, member)
			continue
		}
		tasks = append(tasks, &model.Task{
			TimerID:  timerID,
			RunTimer: unix,
		})
	}
	return tasks, nil
}

func parseStartTime(minuteBucketKey string) (time.Time, error) {
	parts := strings.Split(minuteBucketKey, "_")
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("invalid minuteBucketKey: %s", minuteBucketKey)
	}
	return time.ParseInLocation("2006-01-02 15:04", parts[0], time.Local)
}

func parseMember(member string) (int64, int64, error) {
	parts := strings.Split(member, "_")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid member: %s", member)
	}

	timerID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, 0, err
	}

	unix, err := strconv.ParseInt(parts[1], 10, 64)

	return timerID, unix, err
}
