package util_test

import (
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/pitabwire/util"
)

func TestCustomHandler(t *testing.T) {
	// Create a new logger with the custom handler
	logger := util.NewLogger(t.Context(), util.WithLogLevel(slog.LevelDebug))
	defer logger.Release() // Return to pool when done

	// Log some messages
	logger.Info("This will be logged in JSON format")
	logger.Debug("Debug message in JSON format", "key", "value")

	// output:
	// (JSON-formatted log output)
}

func TestDirectHandlerUsage(t *testing.T) {
	// Create a text handler
	textHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// Create a new logger with the direct handler
	logger := util.NewLogger(t.Context(), util.WithLogHandler(textHandler), util.WithLogStackTrace())
	defer logger.Release() // Return to pool when done

	// Log some messages
	logger.Info("This will be logged in text format")
	logger.Error("This will include a stack trace")

	// output:
	// (Text-formatted log output)
}

// TestCustomOutputWriter tests using a custom output writer.
func TestCustomOutputWriter(t *testing.T) {
	// Create a buffer to capture logs
	var buf io.Writer = os.Stderr

	// Create a new logger with custom output
	logger := util.NewLogger(t.Context(), util.WithLogOutput(buf))
	defer logger.Release() // Return to pool when done

	// Log some messages
	logger.Info("This message will be written to the custom writer")
}
