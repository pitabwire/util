package util_test

import (
	"context"
	"errors"
	"testing"

	"github.com/pitabwire/util"
)

func TestLogs(t *testing.T) {
	ctx := context.Background()
	logger := util.Log(ctx)
	logger.Debug("testing debug logs")
	logger.Info("testing logs")

	err := errors.New("")
	logger.WithError(err).Error("testing errors")

}

func TestStackTraceLogs(t *testing.T) {
	ctx := context.Background()
	defaultLogs := util.DefaultLogOptions()
	defaultLogs.ShowStackTrace = true
	logger := util.NewLogger(ctx, defaultLogs)
	logger.Debug("testing debug logs")
	logger.Info("testing logs")

	err := errors.New("")
	logger.WithError(err).Error("testing errors")
}
