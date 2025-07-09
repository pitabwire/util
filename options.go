package util

import (
	"io"
	"log/slog"
	"time"

	"github.com/lmittmann/tint"
)

// logOptions contains configuration for the logging system.
// This is intentionally kept private to hide implementation details.
type logOptions struct {
	// level defines the minimum log level that will be output
	level slog.Level

	// addSource determines whether source code position should be added to log entries
	addSource bool

	// timeFormat defines how timestamps are formatted in logs
	timeFormat string

	// noColor disables colored output when set to true
	noColor bool

	// showStackTrace enables automatic stack trace printing for Error and Fatal logs
	showStackTrace bool

	// output specifies the destination for log output (defaults to os.Stdout or os.Stderr based on level)
	output io.Writer

	// handler specifies a custom slog.Handler implementation to use
	handler slog.Handler
}

// Option is a function that configures logOptions.
type Option func(*logOptions)

// defaultLogOptions returns a logOptions instance with sensible defaults.
func defaultLogOptions() *logOptions {
	return &logOptions{
		level:          slog.LevelInfo,
		addSource:      false,
		timeFormat:     time.DateTime,
		noColor:        false,
		showStackTrace: false,
	}
}

// defaultHandlerCreator creates the default tint-based colored slog.Handler.
func defaultHandlerCreator(writer io.Writer, opts *logOptions) slog.Handler {
	handlerOptions := &tint.Options{
		AddSource:  opts.addSource,
		Level:      opts.level,
		TimeFormat: opts.timeFormat,
		NoColor:    opts.noColor,
	}

	return tint.NewHandler(writer, handlerOptions)
}

// WithLogLevel sets the log level.
func WithLogLevel(level slog.Level) Option {
	return func(o *logOptions) {
		o.level = level
	}
}

// WithLogAddSource enables or disables source code position in log entries.
func WithLogAddSource(addSource bool) Option {
	return func(o *logOptions) {
		o.addSource = addSource
	}
}

// WithLogTimeFormat sets the time format for log timestamps.
func WithLogTimeFormat(format string) Option {
	return func(o *logOptions) {
		o.timeFormat = format
	}
}

// WithLogNoColor enables or disables colored output.
func WithLogNoColor(noColor bool) Option {
	return func(o *logOptions) {
		o.noColor = noColor
	}
}

// WithLogStackTrace enables automatic stack trace printing.
func WithLogStackTrace() Option {
	return func(o *logOptions) {
		o.showStackTrace = true
	}
}

// WithLogOutput sets the output writer for logs.
func WithLogOutput(output io.Writer) Option {
	return func(o *logOptions) {
		o.output = output
	}
}

// WithLogHandler sets a custom slog.Handler implementation.
func WithLogHandler(handler slog.Handler) Option {
	return func(o *logOptions) {
		o.handler = handler
	}
}

// ParseLevel converts a string to a log.level.
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
