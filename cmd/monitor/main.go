package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"netmonitor/internal/app"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := app.Run(ctx, "config/monitor-net.yaml"); err != nil {
		log.Fatalf("application exited with error: %v", err)
	}
}
