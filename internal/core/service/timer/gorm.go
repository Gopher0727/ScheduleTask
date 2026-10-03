package timer

import (
	"context"

	"gorm.io/gorm"

	"github.com/Gopher0727/ScheduleTask/internal/core/model"
)

type Store struct{ DB *gorm.DB }

func (s *Store) Save(ctx context.Context, timer *model.Timer) error {
	return s.DB.WithContext(ctx).Create(timer).Error
}

func (s *Store) Update(ctx context.Context, timer *model.Timer) error {
	return s.DB.WithContext(ctx).Save(timer).Error
}

func (s *Store) GetByID(ctx context.Context, id int64) (*model.Timer, error) {
	t := &model.Timer{}
	err := s.DB.WithContext(ctx).First(t, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return t, err
}

func (s *Store) GetByStatus(ctx context.Context, status int) ([]*model.Timer, error) {
	var timers []*model.Timer
	err := s.DB.WithContext(ctx).Where("status = ?", status).Find(&timers).Error
	return timers, err
}

func New(db *gorm.DB) *Store { return &Store{DB: db} }
