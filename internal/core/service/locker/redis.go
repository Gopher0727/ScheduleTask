package locker

import (
	"context"
	"fmt"

	redis "github.com/redis/go-redis/v9"
)

var (
	// Acquire or renew in one Redis operation so ownership cannot race expiry.
	lockScript = redis.NewScript(`
local value = redis.call('get', KEYS[1])
if value == ARGV[1] then
  redis.call('expire', KEYS[1], ARGV[2])
  return 1
end
if redis.call('set', KEYS[1], ARGV[1], 'NX', 'EX', ARGV[2]) then return 1 end
return 0`)
)

var (
	unlockScript = redis.NewScript(`if redis.call('get', KEYS[1]) == ARGV[1] then return redis.call('del', KEYS[1]) else return 0 end`)
	expireScript = redis.NewScript(`local v = redis.call('get', KEYS[1]) if not v or v ~= ARGV[1] then return 0 else return redis.call('expire', KEYS[1], ARGV[2]) end`)
)

type rdLock struct {
	client *redis.Client
}

func (l *rdLock) Lock(ctx context.Context, key, token string, expireSeconds int64) (bool, error) {
	if expireSeconds <= 0 || token == "" {
		return false, fmt.Errorf("lock requires a token and positive expiry")
	}
	value, err := lockScript.Run(ctx, l.client, []string{key}, token, expireSeconds).Int64()
	return value == 1, err
}

func (l *rdLock) Unlock(ctx context.Context, key, token string) error {
	return unlockScript.Run(ctx, l.client, []string{key}, token).Err()
}

func (l *rdLock) Expire(ctx context.Context, key, token string, expireSeconds int64) error {
	return expireScript.Run(ctx, l.client, []string{key}, token, expireSeconds).Err()
}

func New(client *redis.Client) *rdLock { return &rdLock{client: client} }
