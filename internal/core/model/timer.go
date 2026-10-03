package model

import "time"

// Timer stores a recurring definition. Status 0 is disabled; status 1 is enabled.
// CreateTime also anchors constant-delay schedules across repeated materialization.
type Timer struct {
	TimerID         int64     `json:"timer_id" db:"timer_id" gorm:"primaryKey;autoIncrement"`
	App             string    `json:"app" db:"app"`
	Name            string    `json:"name" db:"name"`
	Status          int       `json:"status" db:"status"`
	Cron            string    `json:"cron" db:"cron"`
	NotifyHTTPParam string    `json:"notify_http_param" db:"notify_http_param"`
	CreateTime      time.Time `json:"create_time" db:"create_time" gorm:"autoCreateTime"`
	UpdateTime      time.Time `json:"update_time" db:"update_time" gorm:"autoUpdateTime"`
}
