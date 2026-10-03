package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	TaskExecuted = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "scheduletask_task_executed_total", Help: "Total executed tasks"},
		[]string{"status"},
	)
	TaskDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{Name: "scheduletask_task_duration_seconds", Help: "Task execution latency"},
	)
	SchedulerScans = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "scheduletask_scheduler_scans_total", Help: "Total scheduler scans"},
	)
	MigratorRuns = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "scheduletask_migrator_runs_total", Help: "Total migrator runs"},
	)
	LockAcquired = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "scheduletask_lock_acquired_total", Help: "Total lock acquired"},
	)
	LockFailed = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "scheduletask_lock_failed_total", Help: "Total lock acquire failures"},
	)
)

func init() {
	prometheus.MustRegister(TaskExecuted, TaskDuration, SchedulerScans, MigratorRuns, LockAcquired, LockFailed)
}
