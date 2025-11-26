package util_test

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/pitabwire/util"
)

// TestLogs tests basic logging functionality.
func TestLogs(t *testing.T) {
	ctx := t.Context()
	logger := util.NewLogger(ctx)
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

// TestStackTraceLogs tests logging with stack traces.
func TestStackTraceLogs(t *testing.T) {
	ctx := t.Context()
	logger := util.NewLogger(ctx, util.WithLogStackTrace())
	logger.Debug("testing debug logs")
	logger.Info("testing logs")

	err := errors.New("")
	logger.WithError(err).Error("testing errors")
	defer logger.Release()
}

// TestPanicLogs tests panic recovery in logging.
func TestPanicLogs(t *testing.T) {
	ctx := t.Context()
	logger := util.NewLogger(ctx)

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

// BenchmarkLoggerWithField benchmarks the logger WithField method to measure performance.
func BenchmarkLoggerWithField(b *testing.B) {
	ctx := b.Context()
	logger := util.NewLogger(ctx)
	defer logger.Release()

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		l := logger.WithField("key", "value")
		l.Release() // Important to return to the pool
	}
}

// BenchmarkLoggerMultipleWithField benchmarks chaining multiple WithField calls.
func BenchmarkLoggerMultipleWithField(b *testing.B) {
	ctx := b.Context()
	logger := util.NewLogger(ctx)
	defer logger.Release()

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		l := logger.WithField("key1", "value1").
			WithField("key2", "value2").
			WithField("key3", "value3")
		l.Release() // Important to return to the pool
	}
}

// BenchmarkLoggerWithoutPooling simulates the overhead without using pools.
func BenchmarkLoggerWithoutPooling(b *testing.B) {
	ctx := b.Context()
	logger := util.NewLogger(ctx)
	defer logger.Release()

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		// Intentionally creating and dropping references without explicit release
		_ = logger.WithField("key1", "value1").
			WithField("key2", "value2").
			WithField("key3", "value3")
	}
}

// TestMultiHandlerVerification thoroughly verifies that MultiHandler and handlers are not mutually exclusive.
func TestMultiHandlerVerification(t *testing.T) {
	t.Run("IndividualHandlerUsage", testIndividualHandlerUsage)
	t.Run("MultiHandlerFunctionalityViaAPI", testMultiHandlerFunctionalityViaAPI)
	t.Run("HandlerIndependence", testHandlerIndependence)
	t.Run("HandlerExclusiveMode", testHandlerExclusiveMode)
	t.Run("DefaultMultiHandlerBehavior", testDefaultMultiHandlerBehavior)
	t.Run("MultipleHandlersViaMultipleLoggers", testMultipleHandlersViaMultipleLoggers)
}

func testIndividualHandlerUsage(t *testing.T) {
	ctx := t.Context()
	var buf1, buf2 bytes.Buffer
	handler2 := slog.NewJSONHandler(&buf2, &slog.HandlerOptions{Level: slog.LevelInfo})

	logger := util.NewLogger(ctx, util.WithLogOutput(&buf1), util.WithLogHandler(handler2))
	defer logger.Release()

	logger.Info("test message 1")

	// Verify output
	if !strings.Contains(buf1.String(), "test message 1") {
		t.Error("Handler1 did not log message")
	}
	if !strings.Contains(buf2.String(), "test message 1") {
		t.Error("Handler2 did not log message")
	}
}

func testMultiHandlerFunctionalityViaAPI(t *testing.T) {
	ctx := t.Context()
	var buf bytes.Buffer
	handler1 := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})

	// Test that providing a custom handler works with the default MultiHandler
	logger := util.NewLogger(ctx, util.WithLogHandler(handler1))
	defer logger.Release()
	logger.Info("multi test message")

	// The output should go to both the default handler (stderr) and the custom handler (buf)
	// We can verify the custom handler received the message
	if !strings.Contains(buf.String(), "multi test message") {
		t.Error("Custom handler in MultiHandler did not log message")
	}
}

func testHandlerIndependence(t *testing.T) {
	ctx := t.Context()
	var buf bytes.Buffer
	sharedHandler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})

	// Use handler individually
	individualLogger := slog.New(sharedHandler)
	individualLogger.Info("individual message")

	// Use same handler type in util.NewLogger (which creates MultiHandler internally)
	logger := util.NewLogger(ctx, util.WithLogHandler(sharedHandler))
	defer logger.Release()
	logger.Info("multi message")

	output := buf.String()
	if !strings.Contains(output, "individual message") {
		t.Error("Handler did not work individually")
	}
	if !strings.Contains(output, "multi message") {
		t.Error("Handler did not work in MultiHandler")
	}
}

func testHandlerExclusiveMode(t *testing.T) {
	ctx := t.Context()
	var buf bytes.Buffer
	customHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})

	// Test exclusive mode
	exclusiveLogger := util.NewLogger(ctx, util.WithLogHandler(customHandler), util.WithLogHandlerExclusive())
	defer exclusiveLogger.Release()
	exclusiveLogger.Info("exclusive message")

	// Should only output JSON, not the default tinted handler
	output := buf.String()
	if !strings.Contains(output, `"msg":"exclusive message"`) {
		t.Error("Exclusive handler did not work")
	}
	// The default handler would produce colored text output, not JSON
	if strings.Contains(output, "exclusive message") && !strings.Contains(output, `"msg":`) {
		t.Error("Exclusive mode should use only custom handler, not default")
	}
}

func testDefaultMultiHandlerBehavior(t *testing.T) {
	ctx := t.Context()
	var buf bytes.Buffer
	customHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})

	// Default behavior should create MultiHandler with both default and custom handler
	logger := util.NewLogger(ctx, util.WithLogHandler(customHandler))
	defer logger.Release()

	logger.Info("default multi message")

	output := buf.String()
	// Should contain JSON from custom handler
	if !strings.Contains(output, `"msg":"default multi message"`) {
		t.Error("Custom handler in default MultiHandler did not work")
	}
}

func testMultipleHandlersViaMultipleLoggers(t *testing.T) {
	ctx := t.Context()
	var buf1, buf2 bytes.Buffer
	handler1 := slog.NewTextHandler(&buf1, &slog.HandlerOptions{Level: slog.LevelInfo})
	handler2 := slog.NewJSONHandler(&buf2, &slog.HandlerOptions{Level: slog.LevelInfo})

	// Create two loggers with different handlers to simulate MultiHandler behavior
	logger1 := util.NewLogger(ctx, util.WithLogHandler(handler1), util.WithLogHandlerExclusive())
	defer logger1.Release()
	logger2 := util.NewLogger(ctx, util.WithLogHandler(handler2), util.WithLogHandlerExclusive())
	defer logger2.Release()

	logger1.Info("text message")
	logger2.Info("json message")

	// Verify both handlers work independently
	if !strings.Contains(buf1.String(), "text message") {
		t.Error("Text handler did not work")
	}
	if !strings.Contains(buf2.String(), `"msg":"json message"`) {
		t.Error("JSON handler did not work")
	}
}
