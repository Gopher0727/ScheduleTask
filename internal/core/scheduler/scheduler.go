package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/Gopher0727/ScheduleTask/internal/config/core"
	"github.com/Gopher0727/ScheduleTask/internal/core/provider"
	"github.com/Gopher0727/ScheduleTask/internal/logger"
	"github.com/Gopher0727/ScheduleTask/internal/metrics"
	"github.com/Gopher0727/ScheduleTask/internal/utils"
)

type Scheduler struct {
	lock    provider.Locker
	trigger Trigger
	workers sync.WaitGroup
	cfg     core.Scheduler
}

type Trigger interface {
	Work(ctx context.Context, minuteBucketKey string)
}

func New(lock provider.Locker, trigger Trigger, cfg core.Scheduler) *Scheduler {
	return &Scheduler{
		lock:    lock,
		trigger: trigger,
		cfg:     cfg,
	}
}

func (s *Scheduler) Run(ctx context.Context) {
	defer s.workers.Wait()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics.SchedulerScans.Inc()
			s.handleSlice(ctx)
		}
	}
}

func (s *Scheduler) handleSlice(ctx context.Context) {
	cfg := s.cfg

	now := time.Now()
	nowPreMin := now.Add(-1 * time.Minute)

	for i := 0; i < cfg.BucketNum; i++ {
		s.handleBucket(ctx, nowPreMin, i)
		s.handleBucket(ctx, now, i)
	}
}

func (s *Scheduler) handleBucket(ctx context.Context, t time.Time, bucketID int) {
	cfg := s.cfg

	lockToken := utils.GetTokenStr()
	lockKey := utils.GetTimeBucketLockKey(t, bucketID)

	ok, err := s.lock.Lock(ctx, lockKey, lockToken, int64(cfg.TryLockSeconds))
	if err != nil || !ok {
		metrics.LockFailed.Inc()
		return
	}
	metrics.LockAcquired.Inc()

	// A minute worker must not block acquisition of the remaining buckets.
	s.workers.Add(1)
	go func() {
		defer s.workers.Done()
		s.trigger.Work(ctx, utils.GetSliceMsgKey(t, bucketID))
		if err := s.lock.Expire(ctx, lockKey, lockToken, int64(cfg.SuccessExpireSeconds)); err != nil && ctx.Err() == nil {
			logger.L.Infof("scheduler expire lock failed: %v", err)
		}
	}()
}
