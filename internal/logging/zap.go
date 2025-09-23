package logging

import (
	"fmt"

	"github.com/andrewsvn/gophermart-ls/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewZapLogger(config config.LogConfig) (*zap.Logger, error) {
	lvl, err := zap.ParseAtomicLevel(config.Level)
	if err != nil {
		return nil, err
	}

	logConfig := zap.NewProductionConfig()
	logConfig.DisableCaller = true
	logConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logConfig.EncoderConfig.EncodeDuration = zapcore.StringDurationEncoder
	logConfig.Level = lvl

	logger, err := logConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("unable to create zap logger: %w", err)
	}

	return logger, nil
}

func ComponentLogger(l *zap.Logger, name string) *zap.SugaredLogger {
	return l.Sugar().With("component", name)
}

func Sync(l *zap.Logger) {
	_ = l.Sync()
}
