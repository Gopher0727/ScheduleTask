package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	echo "github.com/labstack/echo/v4"
	"google.golang.org/grpc"

	pb "github.com/Gopher0727/ScheduleTask/api/pb"
	"github.com/Gopher0727/ScheduleTask/internal/config"
	"github.com/Gopher0727/ScheduleTask/internal/core/executor"
	"github.com/Gopher0727/ScheduleTask/internal/core/migrator"
	"github.com/Gopher0727/ScheduleTask/internal/core/model"
	"github.com/Gopher0727/ScheduleTask/internal/core/scheduler"
	"github.com/Gopher0727/ScheduleTask/internal/core/service/locker"
	"github.com/Gopher0727/ScheduleTask/internal/core/service/task"
	"github.com/Gopher0727/ScheduleTask/internal/core/service/timer"
	"github.com/Gopher0727/ScheduleTask/internal/core/service/timewheel"
	"github.com/Gopher0727/ScheduleTask/internal/core/trigger"
	"github.com/Gopher0727/ScheduleTask/internal/infra/postgres"
	"github.com/Gopher0727/ScheduleTask/internal/infra/redis"
	"github.com/Gopher0727/ScheduleTask/internal/logger"
	httpServer "github.com/Gopher0727/ScheduleTask/internal/server/http"
	rpcServer "github.com/Gopher0727/ScheduleTask/internal/server/rpc"
)

// Run wires dependencies and owns their lifetime until cancellation or server failure.
func Run(ctx context.Context, configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	startupCtx, startupCancel := context.WithTimeout(ctx, 10*time.Second)
	defer startupCancel()
	db, err := postgres.New(startupCtx, cfg.Postgres)
	if err != nil {
		return fmt.Errorf("PostgreSQL startup: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	redisClient := redis.New(cfg.Redis)
	defer redisClient.Close()
	if err := redisClient.Ping(startupCtx).Err(); err != nil {
		return fmt.Errorf("Redis startup: %w", err)
	}
	if err := db.WithContext(startupCtx).AutoMigrate(&model.Timer{}, &model.Task{}); err != nil {
		return fmt.Errorf("schema migration: %w", err)
	}
	timerStore := timer.New(db)
	taskStore := task.New(db)
	lock := locker.New(redisClient)
	timeWheel := timewheel.New(redisClient)

	// Build adapters only after all dependencies and schemas are ready.
	echoServer := echo.New()

	httpHandler := httpServer.New(timerStore, taskStore)
	httpHandler.SetReadinessCheck(func(ctx context.Context) error {
		if err := sqlDB.PingContext(ctx); err != nil {
			return err
		}
		return redisClient.Ping(ctx).Err()
	})
	httpHandler.RegisterRoutes(echoServer)

	grpcServer := grpc.NewServer()
	rpcHandler := rpcServer.New(timerStore)
	pb.RegisterXTimerServer(grpcServer, rpcHandler)

	t := trigger.New(timeWheel, taskStore, executor.New(taskStore, timerStore), cfg.Trigger, cfg.Scheduler.BucketNum)
	s := scheduler.New(lock, t, cfg.Scheduler)
	m := migrator.New(timerStore, taskStore, timeWheel, lock, cfg.Migrator, cfg.Scheduler.BucketNum)

	httpAddr := fmt.Sprintf(":%d", cfg.Port)
	grpcAddr := fmt.Sprintf(":%d", cfg.Port+1)
	httpListener, err := net.Listen("tcp", httpAddr)
	if err != nil {
		return fmt.Errorf("HTTP listen: %w", err)
	}
	defer httpListener.Close()
	grpcListener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return fmt.Errorf("gRPC listen: %w", err)
	}
	defer grpcListener.Close()
	echoServer.Listener = httpListener
	echoServer.HideBanner = true
	echoServer.Server.ReadHeaderTimeout = 5 * time.Second
	echoServer.Server.IdleTimeout = 60 * time.Second

	serverErrors := make(chan error, 2)
	go func() { serverErrors <- echoServer.Start(httpAddr) }()
	go func() { serverErrors <- grpcServer.Serve(grpcListener) }()
	var workers sync.WaitGroup
	workers.Add(2)
	go func() { defer workers.Done(); s.Run(ctx) }()
	go func() { defer workers.Done(); m.Run(ctx) }()
	logger.L.Infof("HTTP listening on %s; gRPC listening on %s", httpAddr, grpcAddr)

	var serveErr error
	select {
	case <-ctx.Done():
	case serveErr = <-serverErrors:
	}
	logger.L.Info("shutting down server...")
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	grpcDone := make(chan struct{})
	go func() { grpcServer.GracefulStop(); close(grpcDone) }()
	if err := echoServer.Shutdown(shutdownCtx); err != nil {
		echoServer.Close()
	}
	select {
	case <-grpcDone:
	case <-shutdownCtx.Done():
		grpcServer.Stop()
	}
	workersDone := make(chan struct{})
	go func() { workers.Wait(); close(workersDone) }()
	select {
	case <-workersDone:
	case <-shutdownCtx.Done():
		return fmt.Errorf("worker shutdown: %w", shutdownCtx.Err())
	}
	if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) && !errors.Is(serveErr, grpc.ErrServerStopped) {
		return serveErr
	}
	return nil
}
