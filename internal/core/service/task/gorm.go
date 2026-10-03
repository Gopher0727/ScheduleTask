package task

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Gopher0727/ScheduleTask/internal/core/model"
)

type Store struct {
	DB *gorm.DB
}

func (s *Store) BatchSave(ctx context.Context, tasks []*model.Task) error {
	if len(tasks) == 0 {
		return nil
	}
	return s.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "timer_id"}, {Name: "run_timer"}}, DoNothing: true,
	}).CreateInBatches(tasks, 500).Error
}

func (s *Store) GetByTimeRange(ctx context.Context, start, end int64, status int) ([]*model.Task, error) {
	var tasks []*model.Task
	err := s.DB.WithContext(ctx).
		Where("run_timer >= ? AND run_timer <= ? AND status = ?", start, end, status).
		Find(&tasks).Error
	return tasks, err
}

func (s *Store) GetByTimerIDAndRunTime(ctx context.Context, timerID, runTime int64) (*model.Task, error) {
	t := &model.Task{}
	err := s.DB.WithContext(ctx).
		Where("timer_id = ? AND run_timer = ?", timerID, runTime).
		First(t).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return t, err
}

func (s *Store) Update(ctx context.Context, task *model.Task) error {
	return s.DB.WithContext(ctx).Save(task).Error
}

// Claim atomically transitions a pending task to running before sending a callback.
func (s *Store) Claim(ctx context.Context, taskID int64) (bool, error) {
	result := s.DB.WithContext(ctx).Model(&model.Task{}).
		Where("task_id = ? AND status = 0", taskID).Update("status", 3)
	return result.RowsAffected == 1, result.Error
}

func (s *Store) ListByTimerID(ctx context.Context, timerID int64, limit int) ([]*model.Task, error) {
	var tasks []*model.Task
	err := s.DB.WithContext(ctx).Where("timer_id = ? AND run_timer <= ?", timerID, time.Now().UnixMilli()).
		Order("run_timer DESC").Limit(limit).Find(&tasks).Error
	return tasks, err
}

func New(db *gorm.DB) *Store { return &Store{DB: db} }
