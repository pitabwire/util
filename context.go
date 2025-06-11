package util

import (
	"context"
)

// contextKeys is a type alias for string to namespace Context keys per-package.
type contextKeys string

// ctxValueRequestID is the key to extract the request ID for an HTTP request
const ctxValueRequestID = contextKeys("requestid")

// GetRequestID returns the request ID associated with this context, or the empty string
// if one is not associated with this context.
func GetRequestID(ctx context.Context) string {
	id := ctx.Value(ctxValueRequestID)
	if id == nil {
		return ""
	}
	return id.(string)
}
