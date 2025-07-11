package util

import (
	"crypto/rand"
	"math/big"
	"time"

	"github.com/rs/xid"
)

const alphanumerics = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// RandomString generates a cryptographically secure random string of length n.
func RandomString(n int) string {
	if n <= 0 {
		return ""
	}

	b := make([]byte, n)
	for i := range b {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphanumerics))))
		if err != nil {
			panic(err)
		}
		b[i] = alphanumerics[idx.Int64()]
	}
	return string(b)
}

func IDString() string {
	return IDStringWithTime(time.Now())
}

func IDStringWithTime(t time.Time) string {
	return xid.NewWithTime(t).String()
}
