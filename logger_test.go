package util_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/pitabwire/util"
)

func TestLogs(t *testing.T) {
	ctx := context.Background()
	logger := util.NewLogger(ctx, util.DefaultLogOptions())
	logger.Info("test")
	logger.Debug("debugging")
	logger.Error("error occurred")
	logger.Error("error occurred with field", "field", "field-value")

	err := errors.New("")
	logger.WithError(err).Error("testing errors")
	withLog := logger.WithField("g1", "group 1")
	withLog2 := withLog.WithField("g2", "group 2")

	withLog.Info("testing group 1")
	withLog2.Info("testing group 2")

	withLog3 := withLog2.WithField("g3", "group 3")
	withLog2.WithError(err).Error("testing group 2 errors")

	withLog3.Info("testing group 3")

	// Release loggers back to the pool
	defer withLog.Release()
	defer withLog2.Release()
	defer withLog3.Release()
}

func TestStackTraceLogs(t *testing.T) {
	ctx := context.Background()
	defaultLogs := util.DefaultLogOptions()
	defaultLogs.ShowStackTrace = true
	logger := util.NewLogger(ctx, defaultLogs)
	logger.Debug("testing debug logs")
	logger.Info("testing logs")

	err := errors.New("")
	logger.WithError(err).Error("testing errors")
	defer logger.Release()
}

func TestPanicLogs(t *testing.T) {
	ctx := context.Background()
	defaultLogs := util.DefaultLogOptions()
	logger := util.NewLogger(ctx, defaultLogs)

	logger.Info("testing logs")
	defer logger.Release()

	// Set up a deferred function that will recover from the panic
	didPanic := false
	defer func() {
		if r := recover(); r != nil {
			didPanic = true
			// Optional: Check the panic message or value
			// if !strings.Contains(fmt.Sprint(r), "expected panic message") {
			//     t.Errorf("unexpected panic message: %v", r)
			// }
		}

		if !didPanic {
			t.Error("expected Panic() to panic, but it didn't")
		}
	}()

	// Call the function that should panic
	logger.Panic("this should panic")

	// If we get here without panicking, the test will fail
	t.Error("execution continued past panic point")
}

// BenchmarkLoggerWithField benchmarks the logger WithField method to measure performance
func BenchmarkLoggerWithField(b *testing.B) {
	ctx := context.Background()
	logger := util.NewLogger(ctx, util.DefaultLogOptions())
	defer logger.Release()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l := logger.WithField("key", "value")
		l.Release() // Important to return to the pool
	}
}

// BenchmarkLoggerMultipleWithField benchmarks chaining multiple WithField calls
func BenchmarkLoggerMultipleWithField(b *testing.B) {
	ctx := context.Background()
	logger := util.NewLogger(ctx, util.DefaultLogOptions())
	defer logger.Release()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l := logger.WithField("key1", "value1").
			WithField("key2", "value2").
			WithField("key3", "value3")
		l.Release() // Important to return to the pool
	}
}

// BenchmarkLoggerWithoutPooling simulates the overhead without using pools
func BenchmarkLoggerWithoutPooling(b *testing.B) {
	ctx := context.Background()
	logger := util.NewLogger(ctx, util.DefaultLogOptions())
	defer logger.Release()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Intentionally creating and dropping references without explicit release
		_ = logger.WithField("key1", "value1").
			WithField("key2", "value2").
			WithField("key3", "value3")
		// No release call - this shows the cost of GC handling these objects
	}
}

// BenchmarkLogAllocation measures allocation in logging operations
func BenchmarkLogAllocation(b *testing.B) {
	ctx := context.Background()
	logger := util.NewLogger(ctx, util.DefaultLogOptions())
	defer logger.Release()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Typical logging pattern: context with some fields then log
		l := logger.WithField("request_id", fmt.Sprintf("req-%d", i))
		l.Info("Processing request", "index", i)
		l.Release()
	}
}
