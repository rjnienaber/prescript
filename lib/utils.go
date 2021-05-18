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

func CreateLogger() *zap.Logger {
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Println("could not create logger", err.Error())
		// TODO: remove the need for this call
		os.Exit(INTERNAL_ERROR)
	}
	return logger
}
