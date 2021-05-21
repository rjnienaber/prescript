package internal

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
	default:
		return zap.NewNop(), nil // turn logging off for all other values
	}

	config := zap.NewDevelopmentConfig()
	config.Level = zapLevel
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
		logger.zapLogger.Debug(args)
	}

}

func (logger *CustomLogger) Debugf(template string, args ...interface{}) {
	if logger.zapLogger != nil {
		logger.zapLogger.Debugf(template, args)
	}
}

func (logger *CustomLogger) Error(args ...interface{}) {
	if logger.zapLogger != nil {
		logger.zapLogger.Error(args)
	}
}

func (logger *CustomLogger) Info(args ...interface{}) {
	if logger.zapLogger != nil {
		logger.zapLogger.Info(args)
	}
}
