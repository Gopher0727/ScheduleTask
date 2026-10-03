package utils

import (
	"crypto/rand"
	"fmt"
	"time"
)

func GetTokenStr() string {
	// Lease ownership must be unique across processes, not just within one clock.
	return rand.Text()
}

func GetTimeBucketLockKey(t time.Time, bucketID int) string {
	return fmt.Sprintf("time_bucket_lock_%s_%d", t.Format("2006-01-02 15:04"), bucketID)
}

func GetSliceMsgKey(t time.Time, bucketID int) string {
	return fmt.Sprintf("%s_%d", t.Format("2006-01-02 15:04"), bucketID)
}

func UnionTimerIDUnix(timerID, unix int64) string {
	return fmt.Sprintf("%d_%d", timerID, unix)
}

func SplitTimerIDUnix(timerIDUnix string) (int64, int64, error) {
	var timerID, unix int64
	_, err := fmt.Sscanf(timerIDUnix, "%d_%d", &timerID, &unix)
	return timerID, unix, err
}

func TruncateToHour(t time.Time) time.Time {
	return t.Truncate(time.Hour)
}

func GetMigratorLockKey(t time.Time) string {
	return fmt.Sprintf("migrator_lock_%s", t.Format("2006-01-02 15:04"))
}
