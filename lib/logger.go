package lib

import (
	"fmt"
	"go.uber.org/zap"
)

func createLogger() (*zap.Logger, error) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Println("could not create zapLogger", err.Error())
		return nil, err
	}
	return logger, nil
}

type Logger struct {
	zapLogger *zap.SugaredLogger
}

func NewLogger() (Logger, error) {
	logger, err := createLogger()
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
