package logger

import (
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Logger wraps zerolog logger
type Logger struct {
	*zerolog.Logger
}

// New creates a new logger with the specified level
func New(level string) *Logger {
	// Parse log level
	var logLevel zerolog.Level
	switch strings.ToLower(level) {
	case "debug":
		logLevel = zerolog.DebugLevel
	case "info":
		logLevel = zerolog.InfoLevel
	case "warn", "warning":
		logLevel = zerolog.WarnLevel
	case "error":
		logLevel = zerolog.ErrorLevel
	case "fatal":
		logLevel = zerolog.FatalLevel
	case "panic":
		logLevel = zerolog.PanicLevel
	default:
		logLevel = zerolog.InfoLevel
	}

	// Configure zerolog
	zerolog.TimeFieldFormat = time.RFC3339
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	
	logger := zerolog.New(output).
		Level(logLevel).
		With().
		Timestamp().
		Caller().
		Logger()

	return &Logger{&logger}
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() *Logger {
	return &Logger{&log.Logger}
}
