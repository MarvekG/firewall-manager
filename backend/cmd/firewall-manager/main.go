package main

import (
	"log/slog"
	"os"

	"firewall-manager/backend/internal/app"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if err := app.Run(logger); err != nil {
		logger.Error("application stopped", "error", err)
		os.Exit(1)
	}
}
