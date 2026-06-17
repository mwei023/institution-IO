package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Setup configures the global zerolog logger.
func Setup(level, environment string) {
	zerolog.TimeFieldFormat = time.RFC3339

	var l zerolog.Level
	switch level {
	case "debug":
		l = zerolog.DebugLevel
	case "warn":
		l = zerolog.WarnLevel
	case "error":
		l = zerolog.ErrorLevel
	default:
		l = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(l)

	if environment == "development" {
		// Pretty-print in development
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "15:04:05"})
	} else {
		// JSON in production
		log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	}
}

// Get returns the global logger.
func Get() zerolog.Logger {
	return log.Logger
}
