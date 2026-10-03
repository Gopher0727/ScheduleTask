package core

type Scheduler struct {
	BucketNum            int `mapstructure:"bucket_num"`
	TryLockSeconds       int `mapstructure:"try_lock_seconds"`
	SuccessExpireSeconds int `mapstructure:"success_expire_seconds"`
}
