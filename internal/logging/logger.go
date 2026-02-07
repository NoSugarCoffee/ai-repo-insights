package logging

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

// NewLogger creates a new configured logger instance
func NewLogger(level string) zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339

	logLevel := parseLogLevel(level)
	
	logger := zerolog.New(os.Stdout).
		Level(logLevel).
		With().
		Timestamp().
		Logger()

	return logger
}

// parseLogLevel converts string level to zerolog.Level
func parseLogLevel(level string) zerolog.Level {
	switch level {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

// LogError logs an error with context
func LogError(logger zerolog.Logger, err error, context map[string]string) {
	event := logger.Error().Err(err)
	for key, value := range context {
		event = event.Str(key, value)
	}
	event.Msg("error occurred")
}

// LogProgress logs pipeline progress
func LogProgress(logger zerolog.Logger, step string, startTime time.Time, status string) {
	duration := time.Since(startTime)
	logger.Info().
		Str("step", step).
		Dur("duration", duration).
		Str("status", status).
		Msg("pipeline step completed")
}
