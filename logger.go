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

type contextKeyType string

const ctxValueLogger contextKeyType = "logger"

const (
	CallerDepth  = 2
	FileLineAttr = "caller"
)

// ContextWithLogger associates a logger with the context.
func ContextWithLogger(ctx context.Context, logger *LogEntry) context.Context {
	return context.WithValue(ctx, ctxValueLogger, logger)
}

// Log extracts the logger from context or creates a new one.
func Log(ctx context.Context) *LogEntry {
	if v := ctx.Value(ctxValueLogger); v != nil {
		if l, ok := v.(*LogEntry); ok {
			return l
		}
	}
	return NewLogger(ctx)
}

// SLog exposes slog.Logger via context.
func SLog(ctx context.Context) *slog.Logger {
	return Log(ctx).log
}

// LogEntry is a lightweight wrapper with optional stack traces.
type LogEntry struct {
	ctx         context.Context
	log         *slog.Logger
	stackTraces bool
}

var logEntryPool = sync.Pool{
	New: func() interface{} { return new(LogEntry) },
}

// NewLogger constructs a logger. No global side effects.
func NewLogger(ctx context.Context, opts ...Option) *LogEntry {
	options := defaultLogOptions()
	for _, opt := range opts {
		opt(options)
	}

	var out io.Writer
	if options.output != nil {
		out = options.output
	} else if options.level >= slog.LevelError {
		out = os.Stderr
	} else {
		out = os.Stdout
	}

	handler := defaultHandlerCreator(out, options)
	s := slog.New(handler)

	entry := logEntryPool.Get().(*LogEntry)
	entry.ctx = ctx
	entry.log = s
	entry.stackTraces = options.showStackTrace

	return entry
}

// Release returns the entry to the pool.
func (e *LogEntry) Release() {
	if e == nil {
		return
	}
	e.ctx = nil
	e.log = nil
	e.stackTraces = false
	logEntryPool.Put(e)
}

// clone copies a LogEntry efficiently.
func (e *LogEntry) clone() *LogEntry {
	n := logEntryPool.Get().(*LogEntry)
	n.ctx = e.ctx
	n.log = e.log
	n.stackTraces = e.stackTraces
	return n
}

func (e *LogEntry) WithContext(ctx context.Context) *LogEntry {
	n := e.clone()
	n.ctx = ctx
	return n
}

func (e *LogEntry) WithError(err error) *LogEntry {
	return e.With(tint.Err(err))
}

func (e *LogEntry) WithField(key string, value any) *LogEntry {
	return e.With(slog.Any(key, value))
}

func (e *LogEntry) WithFields(fields map[string]any) *LogEntry {
	if len(fields) == 0 {
		return e
	}
	args := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return e.With(args...)
}

func (e *LogEntry) With(args ...any) *LogEntry {
	if len(args) == 0 {
		return e
	}
	n := e.clone()
	n.log = e.log.With(args...)
	return n
}

func (e *LogEntry) ctxOrBackground() context.Context {
	if e.ctx != nil {
		return e.ctx
	}
	return context.Background()
}

func (e *LogEntry) Log(ctx context.Context, level slog.Level, msg string, fields ...any) {
	e.log.Log(ctx, level, msg, fields...)
}

func (e *LogEntry) Logf(ctx context.Context, level slog.Level, format string, args ...interface{}) {
	if e.log.Enabled(ctx, level) {
		e.log.Log(ctx, level, fmt.Sprintf(format, args...))
	}
}

func (e *LogEntry) Trace(msg string, args ...any) {
	e.Debug(msg, args...)
}

func (e *LogEntry) Debug(msg string, args ...any) {
	l := e.withCallerInfo()
	l.DebugContext(e.ctxOrBackground(), msg, args...)
}

func (e *LogEntry) Info(msg string, args ...any) {
	e.log.InfoContext(e.ctxOrBackground(), msg, args...)
}

func (e *LogEntry) Printf(format string, args ...any) {
	e.Logf(e.ctxOrBackground(), slog.LevelInfo, format, args...)
}

func (e *LogEntry) Warn(msg string, args ...any) {
	e.log.WarnContext(e.ctxOrBackground(), msg, args...)
}

func (e *LogEntry) Error(msg string, args ...any) {
	l := e.withCallerInfo()
	ctx := e.ctxOrBackground()

	if e.stackTraces {
		stack := string(debug.Stack())
		msg = fmt.Sprintf("%s\n%s", msg, stack)
	}

	l.ErrorContext(ctx, msg, args...)
}

func (e *LogEntry) Fatal(msg string, args ...any) {
	l := e.withCallerInfo()
	ctx := e.ctxOrBackground()

	if e.stackTraces {
		msg = fmt.Sprintf("%s\n%s", msg, debug.Stack())
	}

	l.ErrorContext(ctx, msg, args...)
	e.Release()
	os.Exit(1)
}

func (e *LogEntry) Panic(msg string, args ...any) {
	l := e.withCallerInfo()
	ctx := e.ctxOrBackground()

	if e.stackTraces {
		msg = fmt.Sprintf("%s\n%s", msg, debug.Stack())
	}

	l.ErrorContext(ctx, msg, args...)
	panic(fmt.Sprintf(msg, args...))
}

func (e *LogEntry) Enabled(ctx context.Context, level slog.Level) bool {
	return e.log.Enabled(ctx, level)
}

func (e *LogEntry) SLog() *slog.Logger { return e.log }

func (e *LogEntry) withCallerInfo() *slog.Logger {
	if _, file, line, ok := runtime.Caller(CallerDepth); ok {
		return e.log.With(slog.String(FileLineAttr, fmt.Sprintf("%s:%d", file, line)))
	}
	return e.log
}

// MultiHandler fans out records to multiple handlers.
type MultiHandler struct {
	handlers []slog.Handler
}

func (m *MultiHandler) extendHandler(h ...slog.Handler) {
	m.handlers = append(m.handlers, h...)
}

func (m *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range m.handlers {
		if err := h.Handle(ctx, r); err != nil {
			return err
		}
	}
	return nil
}

func (m *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	n := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		n[i] = h.WithAttrs(attrs)
	}
	return &MultiHandler{handlers: n}
}

func (m *MultiHandler) WithGroup(name string) slog.Handler {
	n := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		n[i] = h.WithGroup(name)
	}
	return &MultiHandler{handlers: n}
}
