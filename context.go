// Package util provides utility functions and helpers for common operations.
//
//nolint:revive,nolintlint // util is an established package name in this codebase
package util

import (
	"context"
)

// contextKeys is a type alias for string to namespace Context keys per-package.
type contextKeys string

// ctxValueRequestID is the key to extract the request ID for an HTTP request.
const ctxValueRequestID = contextKeys("requestid")

func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, ctxValueRequestID, requestID)
}

// GetRequestID returns the request ID associated with this context, or the empty string
// if one is not associated with this context.
func GetRequestID(ctx context.Context) string {
	id := ctx.Value(ctxValueRequestID)
	if id == nil {
		return ""
	}
	str, ok := id.(string)
	if !ok {
		return ""
	}
	return str
}
