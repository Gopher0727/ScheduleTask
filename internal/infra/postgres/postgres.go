package postgres

import (
	"context"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/Gopher0727/ScheduleTask/internal/config/infra"
)

func New(ctx context.Context, cfg infra.Postgres) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	if err := sqlDB.PingContext(ctx); err != nil {
		sqlDB.Close()
		return nil, err
	}
	return db, nil
}
