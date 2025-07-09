package util_test

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/pitabwire/util"
)

func TestCustomHandler(t *testing.T) {
	// Create a custom handler creator that uses JSON format instead of the default tint handler
	jsonHandlerCreator := func(writer io.Writer, opts *util.LogOptions) slog.Handler {
		return slog.NewJSONHandler(writer, &slog.HandlerOptions{
			Level:     opts.Level,
			AddSource: opts.AddSource,
		})
	}

	// Create options with the custom handler creator
	options := util.DefaultLogOptions().
		WithHandlerCreator(jsonHandlerCreator).
		WithLevel(slog.LevelDebug)

	// Create a new logger with the custom handler
	logger := util.NewLogger(context.Background(), options)
	defer logger.Release() // Return to pool when done

	// Log some messages
	logger.Info("This will be logged in JSON format")
	logger.Debug("Debug message in JSON format", "key", "value")

	// Output:
	// (JSON-formatted log output)
}

func TestDirectHandlerUsage(t *testing.T) {
	// Create a text handler
	textHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// Create options with the handler directly set
	options := util.DefaultLogOptions().
		WithHandler(textHandler).
		WithStackTrace(true)

	// Create a new logger with the direct handler
	logger := util.NewLogger(context.Background(), options)
	defer logger.Release() // Return to pool when done

	// Log some messages
	logger.Info("This will be logged in text format")
	logger.Error("This will include a stack trace")

	// Output:
	// (Text-formatted log output)
}

// TestTelemetryHandler tests using the OpenTelemetry handler
func TestTelemetryHandler(t *testing.T) {
	// Create options with telemetry enabled
	options := util.DefaultLogOptions().
		WithTracing(true)

	// Create a new logger with telemetry
	logger := util.NewLogger(t.Context(), options)
	defer logger.Release() // Return to pool when done

	// Log some messages with trace context
	logger.Info("This message will include OpenTelemetry trace context")
}

// TestCustomOutputWriter tests using a custom output writer
func TestCustomOutputWriter(t *testing.T) {
	// Create a buffer to capture logs
	var buf io.Writer = os.Stderr

	// Create options with custom output
	options := util.DefaultLogOptions().
		WithOutput(buf)

	// Create a new logger with custom output
	logger := util.NewLogger(t.Context(), options)
	defer logger.Release() // Return to pool when done

	// Log some messages
	logger.Info("This message will be written to the custom writer")
}
