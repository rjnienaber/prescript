package utils

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type CustomLogger struct {
	zapLogger *zap.SugaredLogger
}

func createLogger(level string) (*zap.Logger, error) {
	var zapLevel zap.AtomicLevel
	switch level {
	case "debug":
		zapLevel = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	case "info":
		zapLevel = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case "error":
		zapLevel = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	case "none", "":
		return zap.NewNop(), nil
	default:
		// Previously any unrecognised value silently disabled logging, so a
		// typo, or `error` before it was a level, looked exactly like a run
		// with nothing to report.
		return nil, fmt.Errorf("unrecognised log level %q: expected one of none, error, info, debug", level)
	}

	config := zap.NewDevelopmentConfig()
	config.Level = zapLevel
	// Development config attaches a stack trace to every error. They point at
	// the logging call rather than the cause, and bury the message.
	config.DisableStacktrace = true
	logger, err := config.Build()
	if err != nil {
		fmt.Println("could not create zapLogger", err.Error())
		return nil, err
	}
	return logger, nil
}

func NewLogger(level string) (CustomLogger, error) {
	logger, err := createLogger(level)
	if err != nil {
		return CustomLogger{}, err
	}
	return CustomLogger{zapLogger: logger.Sugar()}, nil
}

func (logger *CustomLogger) Close() {
	// sync errors can be ignored: https://github.com/uber-go/zap/issues/328
	err := logger.zapLogger.Sync()
	if err != nil {
		return
	}
}

func (logger *CustomLogger) Debug(args ...interface{}) {
	if logger.zapLogger != nil {
		logger.zapLogger.Debug(args...)
	}

}

func (logger *CustomLogger) Debugf(template string, args ...interface{}) {
	if logger.zapLogger != nil {
		logger.zapLogger.Debugf(template, args...)
	}
}

func (logger *CustomLogger) Error(args ...interface{}) {
	if logger.zapLogger != nil {
		logger.zapLogger.Error(args...)
	}
}

func (logger *CustomLogger) Info(args ...interface{}) {
	if logger.zapLogger != nil {
		logger.zapLogger.Info(args...)
	}
}

func (logger *CustomLogger) Infof(template string, args ...interface{}) {
	if logger.zapLogger != nil {
		logger.zapLogger.Infof(template, args...)
	}
}
