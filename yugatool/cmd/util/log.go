package util

import (
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func GetLogger(module string, debug bool) (logr.Logger, error) {
	level := zapcore.InfoLevel
	if debug {
		level = zapcore.DebugLevel
	}

	zc := zap.NewDevelopmentConfig()

	zc.Level = zap.NewAtomicLevelAt(level)

	if !debug {
		zc.DisableStacktrace = true
	}

	z, err := zc.Build()
	if err != nil {
		return logr.Logger{}, err
	}
	log := zapr.NewLogger(z).WithName(module)
	return log, nil
}
