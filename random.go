package util

import (
	"time"

	"math/rand"

	"github.com/rs/xid"
)

const alphanumerics = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// RandomString generates a pseudo-random string of length n.
func RandomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = alphanumerics[rand.Int63()%int64(len(alphanumerics))]
	}
	return string(b)
}

func IDString() string {
	return IDStringWithTime(time.Now())
}

func IDStringWithTime(t time.Time) string {
	return xid.NewWithTime(t).String()
}
