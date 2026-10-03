package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"

	"github.com/Gopher0727/ScheduleTask/internal/config/core"
	"github.com/Gopher0727/ScheduleTask/internal/config/infra"
)

type Config struct {
	Port int `mapstructure:"port"`

	Redis     infra.Redis    `mapstructure:"redis"`
	Postgres  infra.Postgres `mapstructure:"postgres"`
	Scheduler core.Scheduler `mapstructure:"scheduler"`
	Trigger   core.Trigger   `mapstructure:"trigger"`
	Migrator  core.Migrator  `mapstructure:"migrator"`
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetEnvPrefix("SCHEDULETASK")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}
	var next Config
	if err := v.Unmarshal(&next); err != nil {
		return nil, err
	}
	if err := next.Validate(); err != nil {
		return nil, err
	}
	return &next, nil
}

func (c *Config) Validate() error {
	if c.Port < 1 || c.Port > 65534 {
		return fmt.Errorf("port must be between 1 and 65534 (gRPC uses port + 1)")
	}
	if strings.TrimSpace(c.Postgres.DSN) == "" {
		return fmt.Errorf("postgres.dsn is required")
	}
	if strings.TrimSpace(c.Redis.Addr) == "" || c.Redis.DB < 0 || c.Redis.PoolSize < 1 {
		return fmt.Errorf("redis.addr, non-negative redis.db and positive redis.pool_size are required")
	}
	if c.Postgres.MaxOpenConns < 1 || c.Postgres.MaxIdleConns < 0 || c.Postgres.MaxIdleConns > c.Postgres.MaxOpenConns {
		return fmt.Errorf("invalid postgres connection pool limits")
	}
	if c.Scheduler.BucketNum < 1 || c.Scheduler.BucketNum > 1024 {
		return fmt.Errorf("scheduler.bucket_num must be between 1 and 1024")
	}
	if c.Scheduler.TryLockSeconds < 70 || c.Scheduler.SuccessExpireSeconds < 1 {
		return fmt.Errorf("scheduler lock duration must be at least 70 seconds and success expiry must be positive")
	}
	if c.Trigger.ZRangeGapSeconds < 1 || c.Trigger.ZRangeGapSeconds > 60 {
		return fmt.Errorf("trigger.zrange_gap_seconds must be between 1 and 60")
	}
	if c.Migrator.MigrateStepMinutes < 1 || c.Migrator.MigrateStepMinutes > 1440 || c.Migrator.TryLockMinutes < 1 {
		return fmt.Errorf("migrator window must be between 1 and 1440 minutes and lock duration must be positive")
	}
	return nil
}
