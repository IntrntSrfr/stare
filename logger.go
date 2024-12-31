package stare

import (
	"github.com/intrntsrfr/meido/pkg/mio"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ZapLogger is a wrapper around zap that implements mio.Logger and badger.Logger
type ZapLogger struct {
	log *zap.Logger
}

func newLogger(name string) *ZapLogger {
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	cfg.EncoderConfig.CallerKey = ""
	cfg.EncoderConfig.NameKey = ""
	cfg.EncoderConfig.EncodeTime = zapcore.EpochNanosTimeEncoder
	cfg.Encoding = "console"
	cfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	logger, _ := cfg.Build()

	return &ZapLogger{logger.Named(name)}
}

func (z *ZapLogger) Infof(template string, args ...interface{}) {
	z.log.Sugar().Infof(template, args...)
}

func (z *ZapLogger) Warningf(template string, args ...interface{}) {
	z.log.Sugar().Warnf(template, args...)
}

func (z *ZapLogger) Errorf(template string, args ...interface{}) {
	z.log.Sugar().Errorf(template, args...)
}

func (z *ZapLogger) Debugf(template string, args ...interface{}) {
	z.log.Sugar().Debugf(template, args...)
}

func (z *ZapLogger) Info(msg string, pairs ...interface{}) {
	z.log.Sugar().Infow(msg, pairs...)
}

func (z *ZapLogger) Warn(msg string, pairs ...interface{}) {
	z.log.Sugar().Warnw(msg, pairs...)
}

func (z *ZapLogger) Error(msg string, pairs ...interface{}) {
	z.log.Sugar().Errorw(msg, pairs...)
}

func (z *ZapLogger) Debug(msg string, pairs ...interface{}) {
	z.log.Sugar().Debugw(msg, pairs...)
}

func (z *ZapLogger) Named(name string) mio.Logger {
	return &ZapLogger{z.log.Named(name)}
}
