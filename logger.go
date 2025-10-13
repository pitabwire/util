// Package util provides utility functions and helpers for common operations.
// revive:disable:var-naming
package util

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"runtime/debug"
	"sync"

	"github.com/lmittmann/tint"
)

// contextKeyType is used as a type-safe key for context values.
type contextKeyType string

// ctxValueLogger is the key to extract the LogEntry.
const ctxValueLogger contextKeyType = "logger"

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
	v := ctx.Value(ctxValueLogger)
	if v != nil {
		if logger, ok := v.(*LogEntry); ok {
			return logger
		}
	}

	return NewLogger(ctx)
}

// SLog obtains an slog interface from the log entry in the context.
func SLog(ctx context.Context) *slog.Logger {
	return Log(ctx).SLog()
}

// LogEntry handles logging functionality with immutable chained calls.
type LogEntry struct {
	ctx         context.Context
	log         *slog.Logger
	stackTraces bool
}

//nolint:gochecknoglobals // Pool is necessarily global
var logEntryPool = sync.Pool{
	New: func() interface{} {
		return &LogEntry{}
	},
}

// NewLogger creates a new instance of LogEntry with the provided context and options.
func NewLogger(ctx context.Context, opts ...Option) *LogEntry {
	// Start with default options and apply provided options
	options := defaultLogOptions()
	for _, opt := range opts {
		opt(options)
	}

	// Determine output writer
	var outputWriter io.Writer

	if options.output != nil {
		outputWriter = options.output
	} else {
		if options.level >= slog.LevelError {
			outputWriter = os.Stderr
		} else {
			outputWriter = os.Stdout
		}
	}

	// Create handler - use the specified handler or create one using the handler creator.
	handler := defaultHandlerCreator(outputWriter, options)

	// Create logger
	log := slog.New(handler)
	slog.SetDefault(log)

	// Get a LogEntry from the pool
	entry, ok := logEntryPool.Get().(*LogEntry)
	if !ok {
		// Fallback in case of type assertion failure
		entry = &LogEntry{}
	}

	entry.ctx = ctx
	entry.log = log
	entry.stackTraces = options.showStackTrace

	return entry
}

// Release returns the LogEntry to the pool for reuse.
// Call this when you're done with a LogEntry and won't use it again.
func (e *LogEntry) Release() {
	if e == nil {
		return
	}

	// Reset fields to avoid leaking data
	e.ctx = nil
	e.log = nil
	e.stackTraces = false

	logEntryPool.Put(e)
}

// clone creates a new LogEntry with the same properties as the original.
func (e *LogEntry) clone() *LogEntry {
	if e == nil {
		return NewLogger(context.Background())
	}

	// Get a new entry from the pool
	clone, ok := logEntryPool.Get().(*LogEntry)
	if !ok {
		// Fallback in case of type assertion failure
		clone = &LogEntry{}
	}

	// Copy all fields
	clone.ctx = e.ctx
	clone.log = e.log
	clone.stackTraces = e.stackTraces

	return clone
}

// WithContext returns a new LogEntry with the given context.
func (e *LogEntry) WithContext(ctx context.Context) *LogEntry {
	clone := e.clone()
	clone.ctx = ctx
	return clone
}

// WithError returns a new LogEntry with the error added.
func (e *LogEntry) WithError(err error) *LogEntry {
	return e.With(tint.Err(err))
}

// WithField returns a new LogEntry with the field added.
func (e *LogEntry) WithField(key string, value any) *LogEntry {
	return e.With(slog.Any(key, value))
}

// WithFields returns a new LogEntry with the supplied fields added.
func (e *LogEntry) WithFields(fields map[string]any) *LogEntry {
	var data []any
	for k, v := range fields {
		data = append(data, k, v)
	}
	return e.With(data...)
}

// With returns a new LogEntry with the provided attributes added.
func (e *LogEntry) With(args ...any) *LogEntry {
	// No args, return the same logger
	if len(args) == 0 {
		return e
	}

	clone := e.clone()
	clone.log = clone.log.With(args...)
	return clone
}

// _ctx returns the context or background if nil.
func (e *LogEntry) _ctx() context.Context {
	if e.ctx == nil {
		return context.Background()
	}
	return e.ctx
}

// Log logs a message at the given level.
func (e *LogEntry) Log(ctx context.Context, level slog.Level, msg string, fields ...any) {
	e.log.Log(ctx, level, msg, fields...)
}

// Logf logs a formatted message at the given level.
func (e *LogEntry) Logf(ctx context.Context, level slog.Level, format string, args ...interface{}) {
	if e.Enabled(ctx, level) {
		e.log.Log(ctx, level, fmt.Sprintf(format, args...))
	}
}

// Trace logs a message at debug level (alias for backward compatibility).
func (e *LogEntry) Trace(msg string, args ...any) {
	e.Debug(msg, args...)
}

// Debug logs a message at debug level.
func (e *LogEntry) Debug(msg string, args ...any) {
	log := e.withFileLineNum()
	log.DebugContext(e._ctx(), msg, args...)
}

// Info logs a message at info level.
func (e *LogEntry) Info(msg string, args ...any) {
	e.log.InfoContext(e._ctx(), msg, args...)
}

// Printf logs a formatted message at info level.
func (e *LogEntry) Printf(format string, args ...any) {
	e.Logf(e._ctx(), slog.LevelInfo, format, args...)
}

// Warn logs a message at warn level.
func (e *LogEntry) Warn(msg string, args ...any) {
	e.log.WarnContext(e._ctx(), msg, args...)
}

// Error logs a message at error level.
func (e *LogEntry) Error(msg string, args ...any) {
	log := e.withFileLineNum()

	if e.stackTraces {
		log.ErrorContext(e._ctx(), fmt.Sprintf(" %s\n%s\n", msg, debug.Stack()), args...)
	}

	log.ErrorContext(e._ctx(), msg, args...)
}

// Fatal logs a message at error level and exits with code 1.
func (e *LogEntry) Fatal(msg string, args ...any) {
	log := e.withFileLineNum()

	if e.stackTraces {
		log.ErrorContext(e._ctx(), fmt.Sprintf("%s\n%s", msg, debug.Stack()), args...)
	} else {
		log.ErrorContext(e._ctx(), msg, args...)
	}

	// Release the LogEntry back to the pool before exiting
	e.Release()
	e.Exit(1)
}

// Panic logs a message at error level and panics.
func (e *LogEntry) Panic(msg string, args ...any) {
	log := e.withFileLineNum()

	var panicMsg string
	if e.stackTraces {
		formattedMsg := fmt.Sprintf("%s\n%s", msg, debug.Stack())
		log.ErrorContext(e._ctx(), formattedMsg, args...)
		panicMsg = formattedMsg
	} else {
		log.ErrorContext(e._ctx(), msg, args...)
		panicMsg = msg
	}

	// Format the panic message with args if provided
	if len(args) > 0 {
		panicMsg = fmt.Sprintf(panicMsg+" %v", args)
	}

	panic(panicMsg)
}

// Exit terminates the application with the given code.
func (e *LogEntry) Exit(code int) {
	os.Exit(code)
}

// Enabled returns whether the logger will log at the given level.
func (e *LogEntry) Enabled(ctx context.Context, level slog.Level) bool {
	return e.log.Enabled(ctx, level)
}

// LevelEnabled is an alias for Enabled for backward compatibility.
func (e *LogEntry) LevelEnabled(ctx context.Context, level slog.Level) bool {
	return e.Enabled(ctx, level)
}

// SLog returns the underlying slog.Logger.
func (e *LogEntry) SLog() *slog.Logger {
	return e.log
}

// withFileLineNum adds file and line information to the log entry.
func (e *LogEntry) withFileLineNum() *slog.Logger {
	_, file, line, ok := runtime.Caller(CallerDepth)
	if ok {
		return e.log.With(tint.Attr(FileLineAttr, slog.Any("file", fmt.Sprintf("%s:%d", file, line))))
	}
	return e.log
}

// MultiHandler writes logs to multiple slog handlers (e.g. console + OTel).
type MultiHandler struct {
	handlers []slog.Handler
}

func (m *MultiHandler) extendHandler(h ...slog.Handler) {
	m.handlers = append(m.handlers, h...)
}

func (m *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	var enabled bool
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			enabled = true
		}
	}
	return enabled
}

func (m *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	var err error
	for _, h := range m.handlers {
		err = h.Handle(ctx, r)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	hs := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		hs[i] = h.WithAttrs(attrs)
	}
	return &MultiHandler{hs}
}

func (m *MultiHandler) WithGroup(name string) slog.Handler {
	hs := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		hs[i] = h.WithGroup(name)
	}
	return &MultiHandler{hs}
}
