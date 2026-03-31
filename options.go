// Package util provides utility functions and helpers for common operations.
// revive:disable:var-naming
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

	// format selects the output format: "text" (tint colored) or "json" (structured JSON)
	format string

	// output specifies the destination for log output (defaults to os.Stdout or os.Stderr based on level)
	output io.Writer

	// handler specifies a custom slog.Handler implementation to use
	handler slog.Handler

	// handlerExclusive enforces that only the set handler is utilized
	handlerExclusive bool

	// handlerWrapper wraps the stdout handler (tint or JSON) before it is added to the MultiHandler.
	// Use this to inject middleware such as trace context injection without adding dependencies to util.
	handlerWrapper func(slog.Handler) slog.Handler
}

// Option is a function that configures logOptions.
type Option func(*logOptions)

// defaultLogOptions returns a logOptions instance with sensible defaults.
func defaultLogOptions() *logOptions {
	return &logOptions{
		level:            slog.LevelInfo,
		addSource:        false,
		timeFormat:       time.DateTime,
		noColor:          false,
		showStackTrace:   false,
		format:           "text",
		handlerExclusive: false,
	}
}

// defaultHandlerCreator creates the stdout slog.Handler based on format configuration.
// When format is "json", it uses slog.NewJSONHandler for machine-parseable output.
// Otherwise, it uses the tint handler for human-readable colored output.
func defaultHandlerCreator(writer io.Writer, opts *logOptions) slog.Handler {
	if opts == nil {
		opts = defaultLogOptions()
	}

	if opts.handler != nil {
		if opts.handlerExclusive {
			return opts.handler
		}
	}

	var stdHandler slog.Handler
	if opts.format == "json" {
		stdHandler = slog.NewJSONHandler(writer, &slog.HandlerOptions{
			AddSource: opts.addSource,
			Level:     opts.level,
		})
	} else {
		stdHandler = tint.NewHandler(writer, &tint.Options{
			AddSource:  opts.addSource,
			Level:      opts.level,
			TimeFormat: opts.timeFormat,
			NoColor:    opts.noColor,
		})
	}

	if opts.handlerWrapper != nil {
		stdHandler = opts.handlerWrapper(stdHandler)
	}

	multiHandler := &MultiHandler{handlers: []slog.Handler{stdHandler}}

	if opts.handler != nil {
		multiHandler.extendHandler(opts.handler)
	}

	return multiHandler
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

// WithLogHandlerExclusive sets slog.Handler to be exclusively utilized.
func WithLogHandlerExclusive() Option {
	return func(o *logOptions) {
		o.handlerExclusive = true
	}
}

// WithLogFormat sets the output format. Supported values: "text" (default, tint colored)
// and "json" (structured JSON via slog.JSONHandler for production use).
func WithLogFormat(format string) Option {
	return func(o *logOptions) {
		if format == "json" || format == "text" {
			o.format = format
		}
	}
}

// WithLogHandlerWrapper sets a function that wraps the stdout handler (tint or JSON)
// before it is added to the MultiHandler. This allows injecting handler middleware
// (e.g., trace context injection) without adding dependencies to this package.
func WithLogHandlerWrapper(wrapper func(slog.Handler) slog.Handler) Option {
	return func(o *logOptions) {
		o.handlerWrapper = wrapper
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
