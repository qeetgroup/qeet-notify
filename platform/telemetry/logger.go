package telemetry

import (
	"os"

	"github.com/rs/zerolog"
)

// NewLogger returns a structured zerolog logger; console-pretty in development,
// JSON on stderr in all other environments.
func NewLogger(env string) zerolog.Logger {
	if env == "development" {
		return zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).
			With().Timestamp().Logger()
	}
	return zerolog.New(os.Stderr).With().Timestamp().Logger()
}
