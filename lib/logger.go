package lib

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
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

func NewLogger(level string) (Logger, error) {
	logger, err := createLogger(level)
	if err != nil {
		return Logger{}, err
	}
	return Logger{zapLogger: logger.Sugar()}, nil
}

func (logger *Logger) Close() {
	// sync errors can be ignored: https://github.com/uber-go/zap/issues/328
	logger.zapLogger.Sync()
}

func (logger *Logger) Debug(args ...interface{}) {
	if logger.zapLogger != nil {
		logger.zapLogger.Debug(args)
	}

}

func (logger *Logger) Debugf(template string, args ...interface{}) {
	if logger.zapLogger != nil {
		logger.zapLogger.Debugf(template, args)
	}
}

func (logger *Logger) Error(args ...interface{}) {
	if logger.zapLogger != nil {
		logger.zapLogger.Error(args)
	}
}

func (logger *Logger) Info(args ...interface{}) {
	if logger.zapLogger != nil {
		logger.zapLogger.Info(args)
	}
}
