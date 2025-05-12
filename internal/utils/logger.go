package utils

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Logger levels
const (
	DebugLevel = "debug"
	InfoLevel  = "info"
	WarnLevel  = "warn"
	ErrorLevel = "error"
)

// InitLogger initializes the global logger with proper configuration
func InitLogger(level string, logFile string) {
	// Set appropriate log level
	logLevel := zerolog.InfoLevel
	switch level {
	case DebugLevel:
		logLevel = zerolog.DebugLevel
	case InfoLevel:
		logLevel = zerolog.InfoLevel
	case WarnLevel:
		logLevel = zerolog.WarnLevel
	case ErrorLevel:
		logLevel = zerolog.ErrorLevel
	}
	zerolog.SetGlobalLevel(logLevel)

	// Configure console writer for human-readable output
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}

	// Configure multi-writer if logFile is specified
	var writers []io.Writer
	writers = append(writers, consoleWriter)

	if logFile != "" {
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
		if err != nil {
			log.Error().Err(err).Str("path", logFile).Msg("Failed to open log file")
		} else {
			writers = append(writers, file)
		}
	}

	// Set global logger
	multi := zerolog.MultiLevelWriter(writers...)
	log.Logger = zerolog.New(multi).With().Timestamp().Logger()
}
