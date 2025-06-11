package util

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/lmittmann/tint"
)

// ctxValueLogger is the key to extract the LogEntry.
const ctxValueLogger = contextKeys("logger")

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

// SLog obtains an slog interface from the log entry in the context
func SLog(ctx context.Context) *slog.Logger {
	return Log(ctx).l.log
}

type iLogger struct {
	ctx context.Context
	// Function to exit the application, defaults to `os.Exit()`
	ExitFunc exitFunc

	log         *slog.Logger
	stackTraces bool
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

	il := &iLogger{ctx: ctx, log: log, stackTraces: opts.ShowStackTrace}
	return newLogEntry(il)
}

func (l *iLogger) clone(ctx context.Context) *iLogger {
	sl := *l.log
	return &iLogger{ctx: ctx, log: &sl, stackTraces: l.stackTraces}
}

func (l *iLogger) WithError(err error) {

	l.log = l.log.With(tint.Err(err))
	if l.stackTraces {
		l.log = l.log.With("stacktrace", string(debug.Stack()))
	}
}

func (l *iLogger) WithAttr(attr ...any) {
	l.log = l.log.With(attr...)
}

func (l *iLogger) WithField(key string, value any) {
	l.log = l.log.With(key, value)
}

func (l *iLogger) With(args ...any) {
	l.log = l.log.With(args...)
}

func (l *iLogger) _ctx() context.Context {
	if l.ctx == nil {
		return context.Background()
	}
	return l.ctx
}

func (l *iLogger) Log(ctx context.Context, level slog.Level, msg string, fields ...any) {
	l.log.Log(ctx, level, msg, fields...)
}

func (l *iLogger) Trace(msg string, args ...any) {
	l.Debug(msg, args...)
}

func (l *iLogger) Debug(msg string, args ...any) {
	log := l.withFileLineNum()
	log.DebugContext(l._ctx(), msg, args...)
}

func (l *iLogger) Info(msg string, args ...any) {
	l.log.InfoContext(l._ctx(), msg, args...)
}

func (l *iLogger) Warn(msg string, args ...any) {
	l.log.WarnContext(l._ctx(), msg, args...)
}

func (l *iLogger) Error(msg string, args ...any) {

	log := l.withFileLineNum()

	log.ErrorContext(l._ctx(), msg, args...)

}

func (l *iLogger) Fatal(msg string, args ...any) {
	l.log.ErrorContext(l._ctx(), msg, args...)
	l.Exit(1)
}

func (l *iLogger) Panic(msg string, args ...any) {
	l.log.ErrorContext(l._ctx(), msg, args...)
	panic(msg)
}

func (l *iLogger) Exit(code int) {
	if l.ExitFunc == nil {
		l.ExitFunc = os.Exit
	}
	l.ExitFunc(code)
}

func (l *iLogger) Enabled(ctx context.Context, level slog.Level) bool {
	return l.log.Enabled(ctx, level)
}

func (l *iLogger) withFileLineNum() *slog.Logger {
	_, file, line, ok := runtime.Caller(3)
	if ok {
		return l.log.With(tint.Attr(4, slog.Any("file", fmt.Sprintf("%s:%d", file, line))))
	}
	return l.log
}

// LogEntry Need a type to handle the chained calls
type LogEntry struct {
	l *iLogger
}

func newLogEntry(l *iLogger) *LogEntry {
	return &LogEntry{l: l}
}

type exitFunc func(int)

func (e *LogEntry) LevelEnabled(ctx context.Context, level slog.Level) bool {
	return e.l.Enabled(ctx, level)
}

func (e *LogEntry) WithContext(ctx context.Context) *LogEntry {
	return newLogEntry(e.l.clone(ctx))
}

func (e *LogEntry) Log(ctx context.Context, level slog.Level, msg string, fields ...any) {
	e.l.Log(ctx, level, msg, fields...)
}

func (e *LogEntry) Trace(msg string, args ...any) {
	e.l.Debug(msg, args...)
}

func (e *LogEntry) Debug(msg string, args ...any) {
	e.l.Debug(msg, args...)
}

func (e *LogEntry) Info(msg string, args ...any) {
	e.l.Info(msg, args...)
}

func (e *LogEntry) Printf(format string, args ...any) {
	e.Info(format, args...)
}

func (e *LogEntry) Warn(msg string, args ...any) {
	e.l.Warn(msg, args...)
}

func (e *LogEntry) Error(msg string, args ...any) {
	e.l.Error(msg, args...)
}

func (e *LogEntry) Fatal(msg string, args ...any) {
	e.l.Fatal(msg, args...)

}

func (e *LogEntry) Panic(msg string, args ...any) {
	e.l.Panic(msg, args...)
}

func (e *LogEntry) WithAttr(attr ...any) *LogEntry {
	e.l.WithAttr(attr...)
	return e
}

func (e *LogEntry) WithError(err error) *LogEntry {
	e.l.WithError(err)
	return e
}
func (e *LogEntry) WithField(key string, value any) *LogEntry {
	e.l.WithField(key, value)
	return e
}
func (e *LogEntry) With(args ...any) *LogEntry {
	e.l.With(args...)
	return e
}
