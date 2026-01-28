package logging

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Logger wraps zerolog.Logger with Bulwark-specific configuration
type Logger struct {
	*zerolog.Logger
}

// Config holds logger configuration
type Config struct {
	Level         string
	Format        string // "json" or "console"
	RedactSecrets bool
}

// New creates a new configured logger
func New(cfg Config) *Logger {
	// Parse log level
	level, err := zerolog.ParseLevel(strings.ToLower(cfg.Level))
	if err != nil {
		level = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(level)

	// Configure output
	var output io.Writer = os.Stdout
	if cfg.Format == "console" {
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
	}

	// Create logger
	logger := zerolog.New(output).
		With().
		Timestamp().
		Caller().
		Logger()

	return &Logger{Logger: &logger}
}

// Default returns a logger with default configuration
func Default() *Logger {
	return New(Config{
		Level:         "info",
		Format:        "console",
		RedactSecrets: true,
	})
}

// WithComponent returns a new logger with a component field
func (l *Logger) WithComponent(component string) *Logger {
	logger := l.Logger.With().Str("component", component).Logger()
	return &Logger{Logger: &logger}
}

// WithTarget returns a new logger with target context
func (l *Logger) WithTarget(targetID, targetName string) *Logger {
	logger := l.Logger.With().
		Str("target_id", targetID).
		Str("target_name", targetName).
		Logger()
	return &Logger{Logger: &logger}
}

// WithService returns a new logger with service context
func (l *Logger) WithService(serviceName, image string) *Logger {
	logger := l.Logger.With().
		Str("service", serviceName).
		Str("image", image).
		Logger()
	return &Logger{Logger: &logger}
}

// Init initializes the global logger
func Init(cfg Config) {
	logger := New(cfg)
	log.Logger = *logger.Logger
}
