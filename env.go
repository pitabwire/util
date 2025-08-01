// Package util provides utility functions and helpers for common operations.
//
//nolint:revive,nolintlint // util is an established package name in this codebase
package util

import (
	"os"
)

// GetEnv Obtains the environment key or returns the first fallback value.
func GetEnv(key string, fallback ...string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	if len(fallback) > 0 {
		return fallback[0]
	}

	return ""
}
