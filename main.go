package main

import (
	"bufio"
	"fmt"
	"go.uber.org/zap"
	"io"
	"os"
	"os/exec"
	"strings"
)

func createLogger() *zap.Logger {
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Println("Could not create logger", err.Error())
		os.Exit(2)
	}
	return logger
}

func startCommand(args []string, logger *zap.SugaredLogger) io.ReadCloser {
	appArgs := strings.Join(args[1:], ",")
	cmd := exec.Command(args[0], appArgs)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logger.Error("Failed to capture stdout: ", err)
		os.Exit(2)
	}

	cmd.Stderr = cmd.Stdout

	if err = cmd.Start(); err != nil {
		logger.Error("Failed to start app: ", err)
		os.Exit(2)
	}

	return stdout
}

func main() {
	logger := createLogger()
	// sync errors can be ignored: https://github.com/uber-go/zap/issues/328
	defer logger.Sync()
	sugar := logger.Sugar()

	stdout := startCommand(os.Args[1:], sugar)

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}
}
