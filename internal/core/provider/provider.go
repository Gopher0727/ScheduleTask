package provider

import (
	"context"

	"github.com/Gopher0727/ScheduleTask/internal/core/model"
)

// TimerStore persists recurring timer definitions.
type TimerStore interface {
	Save(ctx context.Context, timer *model.Timer) error
	Update(ctx context.Context, timer *model.Timer) error
	GetByID(ctx context.Context, id int64) (*model.Timer, error)
	GetByStatus(ctx context.Context, status int) ([]*model.Timer, error)
}

// TaskStore persists execution state; Claim must be an atomic pending-to-running transition.
type TaskStore interface {
	Claim(ctx context.Context, taskID int64) (bool, error)
	ListByTimerID(ctx context.Context, timerID int64, limit int) ([]*model.Task, error)
	BatchSave(ctx context.Context, tasks []*model.Task) error
	GetByTimeRange(ctx context.Context, start, end int64, status int) ([]*model.Task, error)
	GetByTimerIDAndRunTime(ctx context.Context, timerID, runTime int64) (*model.Task, error)
	Update(ctx context.Context, task *model.Task) error
}

// Locker grants expiring ownership leases; mutation must check the owner token.
type Locker interface {
	Lock(ctx context.Context, key, token string, expireSeconds int64) (bool, error)
	Unlock(ctx context.Context, key, token string) error
	Expire(ctx context.Context, key, token string, expireSeconds int64) error
}

// TimeWheel indexes occurrences by due time. Its contents are disposable, not authoritative.
type TimeWheel interface {
	Add(ctx context.Context, key, member string, score float64) error
	RangeByScore(ctx context.Context, key string, min, max float64) ([]string, error)
	Remove(ctx context.Context, key string, members ...string) error
}
