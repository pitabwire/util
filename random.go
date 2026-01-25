// Package util provides utility functions and helpers for common operations.
// revive:disable:var-naming
package util

import (
	"crypto/rand"
	"math/big"
	"time"

	"github.com/rs/xid"
)

const (
	alphanumerics = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	numerics      = "0123456789"
)

// RandomString generates a cryptographically secure random string of length n
// using the provided character set.
func RandomString(n int, charset string) string {
	if n <= 0 {
		return ""
	}

	maxLen := big.NewInt(int64(len(charset)))
	b := make([]byte, n)

	for i := 0; i < n; i++ {
		idx, err := rand.Int(rand.Reader, maxLen)
		if err != nil {
			panic(err)
		}
		b[i] = charset[idx.Int64()]
	}

	return string(b)
}

// RandomAlphaNumericString generates a cryptographically secure alphanumeric string.
func RandomAlphaNumericString(n int) string {
	return RandomString(n, alphanumerics)
}

// RandomNumericString generates a cryptographically secure numeric string.
func RandomNumericString(n int) string {
	return RandomString(n, numerics)
}

func IDString() string {
	return IDStringWithTime(time.Now())
}

func IDStringWithTime(t time.Time) string {
	return xid.NewWithTime(t).String()
}
