package util

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/lmittmann/tint"
)

// ctxValueLogger is the key to extract the LogEntry.
const ctxValueLogger = contextKeys("logger")

const (
	CallerDepth  = 3
	FileLineAttr = 4
)

// ContextWithLogger pushes a LogEntry instance into the supplied context for easier propagation.
func ContextWithLogger(ctx context.Context, logger *LogEntry) context.Context {
	return context.WithValue(ctx, ctxValueLogger, logger)
}

// Log obtains a service instance being propagated through the context.
func Log(ctx context.Context) *LogEntry {
	logEntry, ok := ctx.Value(ctxValueLogger).(*LogEntry)
	if ok {
		return logEntry
	}

	return NewLogger(ctx, DefaultLogOptions())
}

// SLog obtains an slog interface from the log entry in the context.
func SLog(ctx context.Context) *slog.Logger {
	return Log(ctx).SLog()
}

// LogEntry handles logging functionality with immutable chained calls
type LogEntry struct {
	ctx         context.Context
	log         *slog.Logger
	stackTraces bool
}

// logEntryPool maintains a pool of LogEntry objects to reduce GC pressure
var logEntryPool = sync.Pool{
	New: func() interface{} {
		return &LogEntry{}
	},
}

type LogOptions struct {
	*slog.HandlerOptions
	PrintFormat    string
	TimeFormat     string
	NoColor        bool
	ShowStackTrace bool
}

func DefaultLogOptions() *LogOptions {
	return &LogOptions{
		HandlerOptions: &slog.HandlerOptions{
			AddSource: false,
			Level:     slog.LevelInfo,
		},
		PrintFormat:    "",
		TimeFormat:     time.DateTime,
		NoColor:        false,
		ShowStackTrace: false,
	}
}

// ParseLevel converts a string to a log.Level.
// It is case-insensitive.
// Returns an error if the string does not match a known level.
func ParseLevel(levelStr string) (slog.Level, error) {
	switch strings.ToLower(levelStr) {
	case "debug", "trace":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning": // Add "warning" as an alias for "warn" if desired
		return slog.LevelWarn, nil
	case "error", "fatal", "panic":
		return slog.LevelError, nil
	default:
		// Default to Info or return an error for unrecognized strings
		return slog.LevelInfo, fmt.Errorf("unknown log level: %q", levelStr)
	}
}

// NewLogger creates a new instance of LogEntry with the provided context and options
func NewLogger(ctx context.Context, opts *LogOptions) *LogEntry {
	logLevel := opts.Level.Level()
	outputWriter := os.Stdout
	if logLevel >= slog.LevelError {
		outputWriter = os.Stderr
	}

	handlerOptions := &tint.Options{
		AddSource:  opts.AddSource,
		Level:      logLevel,
		TimeFormat: opts.TimeFormat,
		NoColor:    opts.NoColor,
	}

	tintHandler := tint.NewHandler(outputWriter, handlerOptions)
	log := slog.New(tintHandler)
	slog.SetDefault(log)

	// Get a LogEntry from the pool
	entry := logEntryPool.Get().(*LogEntry)
	entry.ctx = ctx
	entry.log = log
	entry.stackTraces = opts.ShowStackTrace
	
	return entry
}

// Release returns the LogEntry to the pool for reuse
// Call this when you're done with a LogEntry and won't use it again
func (e *LogEntry) Release() {
	if e == nil {
		return
	}

	// Reset fields before returning to pool
	e.ctx = nil
	e.log = nil
	e.stackTraces = false

	logEntryPool.Put(e)
}

// clone creates a new LogEntry with the same properties as the original
func (e *LogEntry) clone() *LogEntry {
	if e == nil {
		return NewLogger(context.Background(), DefaultLogOptions())
	}

	// Get a new entry from the pool
	clone := logEntryPool.Get().(*LogEntry)
	clone.ctx = e.ctx
	clone.log = e.log
	clone.stackTraces = e.stackTraces

	return clone
}

// WithContext returns a new LogEntry with the given context
func (e *LogEntry) WithContext(ctx context.Context) *LogEntry {
	clone := e.clone()
	clone.ctx = ctx
	return clone
}

// WithError returns a new LogEntry with the error added
func (e *LogEntry) WithError(err error) *LogEntry {
	return e.With(tint.Err(err))
}

// WithField returns a new LogEntry with the field added
func (e *LogEntry) WithField(key string, value any) *LogEntry {
	return e.With(key, value)
}

// With returns a new LogEntry with the provided attributes added
func (e *LogEntry) With(args ...any) *LogEntry {
	// No args, return the same logger
	if len(args) == 0 {
		return e
	}

	clone := e.clone()
	clone.log = clone.log.With(args...)
	return clone
}

// _ctx returns the context or background if nil
func (e *LogEntry) _ctx() context.Context {
	if e.ctx == nil {
		return context.Background()
	}
	return e.ctx
}

// Log logs a message at the given level
func (e *LogEntry) Log(ctx context.Context, level slog.Level, msg string, fields ...any) {
	e.log.Log(ctx, level, msg, fields...)
}

// Logf logs a formatted message at the given level
func (e *LogEntry) Logf(ctx context.Context, level slog.Level, format string, args ...interface{}) {
	if e.Enabled(ctx, level) {
		e.log.Log(ctx, level, fmt.Sprintf(format, args...))
	}
}

// Trace logs a message at debug level (alias for backward compatibility)
func (e *LogEntry) Trace(msg string, args ...any) {
	e.Debug(msg, args...)
}

// Debug logs a message at debug level
func (e *LogEntry) Debug(msg string, args ...any) {
	log := e.withFileLineNum()
	log.DebugContext(e._ctx(), msg, args...)
}

// Info logs a message at info level
func (e *LogEntry) Info(msg string, args ...any) {
	e.log.InfoContext(e._ctx(), msg, args...)
}

// Printf logs a formatted message at info level
func (e *LogEntry) Printf(format string, args ...any) {
	e.Logf(e._ctx(), slog.LevelInfo, format, args...)
}

// Warn logs a message at warn level
func (e *LogEntry) Warn(msg string, args ...any) {
	e.log.WarnContext(e._ctx(), msg, args...)
}

// Error logs a message at error level
func (e *LogEntry) Error(msg string, args ...any) {
	log := e.withFileLineNum()

	if e.stackTraces {
		log.ErrorContext(e._ctx(), fmt.Sprintf(" %s\n%s\n", msg, debug.Stack()), args...)
	}

	log.ErrorContext(e._ctx(), msg, args...)
}

// Fatal logs a message at error level and exits with code 1
func (e *LogEntry) Fatal(msg string, args ...any) {
	log := e.withFileLineNum()

	if e.stackTraces {
		log.ErrorContext(e._ctx(), fmt.Sprintf(" %s\n%s\n", msg, debug.Stack()), args...)
	}
	e.log.ErrorContext(e._ctx(), msg, args...)
	e.Exit(1)
}

// Panic logs a message and panics
func (e *LogEntry) Panic(msg string, _ ...any) {
	panic(fmt.Sprintf(" %s\n%s\n", msg, debug.Stack()))
}

// Exit terminates the application with the given code
func (e *LogEntry) Exit(code int) {
	os.Exit(code)
}

// Enabled returns whether the logger will log at the given level
func (e *LogEntry) Enabled(ctx context.Context, level slog.Level) bool {
	return e.log.Enabled(ctx, level)
}

// LevelEnabled is an alias for Enabled for backward compatibility
func (e *LogEntry) LevelEnabled(ctx context.Context, level slog.Level) bool {
	return e.Enabled(ctx, level)
}

// SLog returns the underlying slog.Logger
func (e *LogEntry) SLog() *slog.Logger {
	return e.log
}

// withFileLineNum adds file and line information to the log entry
func (e *LogEntry) withFileLineNum() *slog.Logger {
	_, file, line, ok := runtime.Caller(CallerDepth)
	if ok {
		return e.log.With(tint.Attr(FileLineAttr, slog.Any("file", fmt.Sprintf("%s:%d", file, line))))
	}
	return e.log
}
