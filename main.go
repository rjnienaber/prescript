package main

import (
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
	defer func() {
		if err := logger.Sync(); err != nil {
			fmt.Println("Could not sync logger on exit", err.Error())
		}
	}()
	sugar := logger.Sugar()

	stdout := startCommand(os.Args[1:], sugar)

	for {
		tmp := make([]byte, 1024)
		_, err := stdout.Read(tmp)
		fmt.Print(string(tmp))
		if err != nil {
			if err != io.EOF {
				sugar.Error("Error reading stdout: ", err)
				os.Exit(1)
			}

			break
		}
	}
}
