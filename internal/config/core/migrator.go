package core

type Migrator struct {
	MigrateStepMinutes int `mapstructure:"migrate_step_minutes"`
	TryLockMinutes     int `mapstructure:"try_lock_minutes"`
}
