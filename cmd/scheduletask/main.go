package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/Gopher0727/ScheduleTask/internal/app"
	"github.com/Gopher0727/ScheduleTask/internal/logger"
)

func main() {
	configPath := flag.String("config", "config.toml", "path to TOML configuration")
	flag.Parse()
	logger.Init("info")
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := app.Run(ctx, *configPath); err != nil {
		logger.L.Errorf("service failed: %v", err)
		logger.Sync()
		os.Exit(1)
	}
	logger.Sync()
}
