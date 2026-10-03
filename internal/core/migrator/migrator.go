package migrator

import (
	"context"
	"fmt"
	"time"

	cron "github.com/robfig/cron/v3"

	"github.com/Gopher0727/ScheduleTask/internal/config/core"
	"github.com/Gopher0727/ScheduleTask/internal/core/model"
	"github.com/Gopher0727/ScheduleTask/internal/core/provider"
	"github.com/Gopher0727/ScheduleTask/internal/logger"
	"github.com/Gopher0727/ScheduleTask/internal/metrics"
	"github.com/Gopher0727/ScheduleTask/internal/utils"
)

type Migrator struct {
	timerStore provider.TimerStore
	taskStore  provider.TaskStore
	timeWheel  provider.TimeWheel
	lock       provider.Locker
	cfg        core.Migrator
	bucketNum  int
}

func New(timerStore provider.TimerStore,
	taskStore provider.TaskStore,
	timeWheel provider.TimeWheel,
	lock provider.Locker,
	cfg core.Migrator, bucketNum int,
) *Migrator {
	return &Migrator{
		timerStore: timerStore,
		taskStore:  taskStore,
		timeWheel:  timeWheel,
		lock:       lock,
		cfg:        cfg, bucketNum: bucketNum,
	}
}

func (m *Migrator) Run(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	m.work(ctx)
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.work(ctx)
		}
	}
}

func (m *Migrator) work(ctx context.Context) {
	cfg := m.cfg

	now := time.Now()
	startMinute := now.Truncate(time.Minute)
	lockToken := utils.GetTokenStr()
	lockKey := utils.GetMigratorLockKey(startMinute)

	ok, err := m.lock.Lock(ctx, lockKey, lockToken, int64(cfg.TryLockMinutes)*60)
	if err != nil || !ok {
		logger.L.Infof("migrator get lock failed: %s", lockKey)
		return
	}

	metrics.MigratorRuns.Inc()
	m.migrate(ctx)
}

func (m *Migrator) migrate(ctx context.Context) {
	timers, err := m.timerStore.GetByStatus(ctx, 1)
	if err != nil {
		logger.L.Infof("GetByStatus error: %v", err)
		return
	}

	for _, timer := range timers {
		if ctx.Err() != nil {
			return
		}
		if err := m.migrateTimer(ctx, timer); err != nil {
			logger.L.Infof("migrateTimer error: %v, timerID: %d", err, timer.TimerID)
		}
	}
}

func (m *Migrator) migrateTimer(ctx context.Context, timer *model.Timer) error {
	if timer.Status != 1 {
		return nil
	}

	now := time.Now()
	end := now.Add(time.Duration(2*m.cfg.MigrateStepMinutes) * time.Minute)
	executeTimes, err := executionTimes(ctx, timer, now, end)
	if err != nil {
		return err
	}

	if len(executeTimes) == 0 {
		return nil
	}

	tasks := m.batchTasksFromTimer(timer, executeTimes)

	if err := m.taskStore.BatchSave(ctx, tasks); err != nil {
		return err
	}

	for _, task := range tasks {
		member := utils.UnionTimerIDUnix(task.TimerID, task.RunTimer)
		score := float64(task.RunTimer)
		key := m.getTableName(task)
		if err := m.timeWheel.Add(ctx, key, member, score); err != nil {
			logger.L.Infof("timeWheel.Add error: %v", err)
		}
	}

	return nil
}

func (m *Migrator) batchTasksFromTimer(timer *model.Timer, executeTimes []int64) []*model.Task {
	tasks := make([]*model.Task, 0, len(executeTimes))
	for _, runTimer := range executeTimes {
		tasks = append(tasks, &model.Task{
			App:      timer.App,
			TimerID:  timer.TimerID,
			RunTimer: runTimer,
			Status:   0,
		})
	}
	return tasks
}

func (m *Migrator) getTableName(task *model.Task) string {
	bucketNum := m.bucketNum
	index := task.TimerID % int64(bucketNum)
	t := time.UnixMilli(task.RunTimer)
	return t.Format("2006-01-02 15:04") + "_" + fmt.Sprintf("%d", index)
}

// executionTimes preserves interval phase across repeated materializations.
func executionTimes(ctx context.Context, timer *model.Timer, now, end time.Time) ([]int64, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	schedule, err := cron.ParseStandard(timer.Cron)
	if err != nil {
		return nil, err
	}
	next := schedule.Next(now)
	if interval, ok := schedule.(cron.ConstantDelaySchedule); ok {
		anchor := timer.CreateTime.Truncate(time.Second)
		if anchor.IsZero() {
			return nil, fmt.Errorf("interval timer requires a creation timestamp")
		}
		steps := now.Sub(anchor)/interval.Delay + 1
		if steps < 1 {
			steps = 1
		}
		next = anchor.Add(steps * interval.Delay)
	}
	var times []int64
	for !next.IsZero() && next.Before(end) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if len(times) >= 10000 {
			return nil, fmt.Errorf("materialization exceeds 10000 occurrences; reduce migrate_step_minutes")
		}
		times = append(times, next.UnixMilli())
		following := schedule.Next(next)
		if following.IsZero() {
			break
		}
		if !following.After(next) {
			return nil, fmt.Errorf("cron schedule did not advance")
		}
		next = following
	}
	return times, nil
}
