package utils

import (
	"log/slog"
	"os"
)

var (
	InfoLog  *slog.Logger
	ErrorLog *slog.Logger
)

// InicializarLogger configura los loggers globales
func InicializarLogger(logLevel string, moduleName string) {
	var level slog.Level

	switch logLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	logger := slog.New(handler).With("modulo", moduleName)

	InfoLog = logger
	ErrorLog = logger
}
