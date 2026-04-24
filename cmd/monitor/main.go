package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"netmonitor/internal/app"
	"netmonitor/internal/logging"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	loggerCloser, err := logging.Setup()
	if err != nil {
		log.Fatalf("setup logger: %v", err)
	}

	exitCode := 0
	if err := app.Run(ctx, "config/monitor-net.yaml"); err != nil {
		log.Printf("application exited with error: %v", err)
		exitCode = 1
	}

	cancel()
	if err := loggerCloser.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "close logger: %v\n", err)
		if exitCode == 0 {
			exitCode = 1
		}
	}
	os.Exit(exitCode)
}
