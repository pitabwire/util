package util_test

import (
	"errors"
	"testing"

	"github.com/pitabwire/util"
)

func TestLogs(t *testing.T) {
	ctx := t.Context()
	logger := util.Log(ctx)
	logger.Debug("testing debug logs")
	logger.Info("testing logs")

	err := errors.New("")
	logger.WithError(err).Error("testing errors")
}

func TestStackTraceLogs(t *testing.T) {
	ctx := t.Context()
	defaultLogs := util.DefaultLogOptions()
	defaultLogs.ShowStackTrace = true
	logger := util.NewLogger(ctx, defaultLogs)
	logger.Debug("testing debug logs")
	logger.Info("testing logs")

	err := errors.New("")
	logger.WithError(err).Error("testing errors")
}
