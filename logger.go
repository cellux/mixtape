package main

import (
	"fmt"
	"log/slog"
	"os"
)

var logger *slog.Logger

func ResolveLogLevel(level string) (slog.Level, error) {
	switch level {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("invalid log level: %s", level)
	}
}

func InitLogger(level string) error {
	logLevel, err := ResolveLogLevel(level)
	if err != nil {
		return err
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})
	logger = slog.New(handler)
	return nil
}
