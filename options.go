package util

import (
	"io"
	"log/slog"
	"time"

	"github.com/lmittmann/tint"
)

// HandlerCreator is a function type that creates and returns an slog.Handler based on configuration.
type HandlerCreator func(writer io.Writer, options *LogOptions) slog.Handler

// LogOptions contains configuration for the logging system.
type LogOptions struct {
	// Level defines the minimum log level that will be output
	Level slog.Level

	// AddSource determines whether source code position should be added to log entries
	AddSource bool

	// TimeFormat defines how timestamps are formatted in logs
	TimeFormat string

	// NoColor disables colored output when set to true
	NoColor bool

	// ShowStackTrace enables automatic stack trace printing for Error and Fatal logs
	ShowStackTrace bool

	// EnableTracing enables OpenTelemetry trace context propagation
	EnableTracing bool

	// Output specifies the destination for log output (defaults to os.Stdout or os.Stderr based on level)
	Output io.Writer

	// Handler specifies a custom slog.Handler implementation to use
	Handler slog.Handler

	// HandlerCreator is a function that creates a handler (used if Handler is not set)
	HandlerCreator HandlerCreator
}

// DefaultLogOptions returns a LogOptions instance with sensible defaults.
func DefaultLogOptions() *LogOptions {
	return &LogOptions{
		Level:          slog.LevelInfo,
		AddSource:      false,
		TimeFormat:     time.DateTime,
		NoColor:        false,
		ShowStackTrace: false,
		HandlerCreator: DefaultHandlerCreator,
	}
}

// DefaultHandlerCreator creates the default tint-based colored slog.Handler.
func DefaultHandlerCreator(writer io.Writer, opts *LogOptions) slog.Handler {
	handlerOptions := &tint.Options{
		AddSource:  opts.AddSource,
		Level:      opts.Level,
		TimeFormat: opts.TimeFormat,
		NoColor:    opts.NoColor,
	}

	return tint.NewHandler(writer, handlerOptions)
}

// WithLevel returns a new LogOptions with the specified level.
func (o *LogOptions) WithLevel(level slog.Level) *LogOptions {
	clone := *o
	clone.Level = level
	return &clone
}

// WithAddSource returns a new LogOptions with the AddSource option set.
func (o *LogOptions) WithAddSource(addSource bool) *LogOptions {
	clone := *o
	clone.AddSource = addSource
	return &clone
}

// WithTimeFormat returns a new LogOptions with the specified time format.
func (o *LogOptions) WithTimeFormat(format string) *LogOptions {
	clone := *o
	clone.TimeFormat = format
	return &clone
}

// WithNoColor returns a new LogOptions with the NoColor option set.
func (o *LogOptions) WithNoColor(noColor bool) *LogOptions {
	clone := *o
	clone.NoColor = noColor
	return &clone
}

// WithStackTrace returns a new LogOptions with the ShowStackTrace option set.
func (o *LogOptions) WithStackTrace(showStackTrace bool) *LogOptions {
	clone := *o
	clone.ShowStackTrace = showStackTrace
	return &clone
}

// WithTracing returns a new LogOptions with the EnableTracing option set.
func (o *LogOptions) WithTracing(enableTracing bool) *LogOptions {
	clone := *o
	clone.EnableTracing = enableTracing
	return &clone
}

// WithOutput returns a new LogOptions with the specified output writer.
func (o *LogOptions) WithOutput(output io.Writer) *LogOptions {
	clone := *o
	clone.Output = output
	return &clone
}

// WithHandler returns a new LogOptions with the specified handler.
func (o *LogOptions) WithHandler(handler slog.Handler) *LogOptions {
	clone := *o
	clone.Handler = handler
	return &clone
}

// WithHandlerCreator returns a new LogOptions with the specified handler creator function.
func (o *LogOptions) WithHandlerCreator(creator HandlerCreator) *LogOptions {
	clone := *o
	clone.HandlerCreator = creator
	return &clone
}

// ParseLevel converts a string to a log.Level.
// It is case-insensitive.
// Returns an error if the string does not match a known level.
func ParseLevel(levelStr string) (slog.Level, error) {
	switch levelStr {
	case "debug", "DEBUG", "Debug", "trace", "TRACE", "Trace":
		return slog.LevelDebug, nil
	case "info", "INFO", "Info":
		return slog.LevelInfo, nil
	case "warn", "WARN", "Warn", "warning", "WARNING", "Warning":
		return slog.LevelWarn, nil
	case "error", "ERROR", "Error", "fatal", "FATAL", "Fatal", "panic", "PANIC", "Panic":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, nil
	}
}
