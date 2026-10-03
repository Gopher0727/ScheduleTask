package model

import "time"

// Task represents one scheduled occurrence. Status: 0 pending, 1 succeeded,
// 2 failed, 3 running. Running claims are not automatically retried after crashes.
type Task struct {
	TaskID     int64     `json:"task_id" db:"task_id" gorm:"primaryKey;autoIncrement"`
	App        string    `json:"app" db:"app"`
	TimerID    int64     `json:"timer_id" db:"timer_id" gorm:"uniqueIndex:idx_task_execution"`
	Output     string    `json:"output" db:"output"`
	RunTimer   int64     `json:"run_timer" db:"run_timer" gorm:"uniqueIndex:idx_task_execution;index:idx_task_due"`
	CostTime   int       `json:"cost_time" db:"cost_time"`
	Status     int       `json:"status" db:"status"`
	CreateTime time.Time `json:"create_time" db:"create_time" gorm:"autoCreateTime"`
	UpdateTime time.Time `json:"update_time" db:"update_time" gorm:"autoUpdateTime"`
}
