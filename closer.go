// Package util provides utility functions and helpers for common operations.
//nolint:revive // util is an established package name in this codebase
package util

import (
	"context"
	"io"
)

// CloseAndLogOnError Closes io.Closer and logs the error if any with the messages supplied.
func CloseAndLogOnError(ctx context.Context, closer io.Closer, message ...string) {
	if closer == nil {
		return
	}
	err := closer.Close()
	if err != nil && len(message) > 0 {
		Log(ctx).WithError(err).Error(message[0])
	}
}
