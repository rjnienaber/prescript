package lib

import (
	"fmt"
	"go.uber.org/zap"
	"os"
)

const (
	SUCCESS = iota
	CLI_ERROR
	INTERNAL_ERROR
	USER_ERROR
)

func ProcessError(err error, logger *zap.SugaredLogger, message string) {
	if err != nil {
		logger.Error(message+": ", err)
		os.Exit(INTERNAL_ERROR)
	}
}

func CreateLogger() *zap.Logger {
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Println("could not create logger", err.Error())
		os.Exit(INTERNAL_ERROR)
	}
	return logger
}
