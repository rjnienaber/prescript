package main

import (
	"bufio"
	"fmt"
	"go.uber.org/zap"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

func processError(err error, logger *zap.SugaredLogger, message string) {
	if err != nil {
		logger.Error(message+": ", err)
		os.Exit(2)
	}
}

func createLogger() *zap.Logger {
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Println("Could not create logger", err.Error())
		os.Exit(2)
	}
	return logger
}

func startCommand(args []string, logger *zap.SugaredLogger) (io.ReadCloser, io.WriteCloser, *os.Process) {
	appArgs := strings.Join(args[1:], ",")
	cmd := exec.Command(args[0], appArgs)
	stdout, err := cmd.StdoutPipe()
	processError(err, logger, "Failed to capture stdout")

	cmd.Stderr = cmd.Stdout

	stdin, err := cmd.StdinPipe()
	processError(err, logger, "Failed to capture stdin")

	err = cmd.Start()
	processError(err, logger, "Failed to start app")

	return stdout, stdin, cmd.Process
}

func main() {
	zapLogger := createLogger()
	// sync errors can be ignored: https://github.com/uber-go/zap/issues/328
	defer zapLogger.Sync()
	logger := zapLogger.Sugar()

	stdout, stdin, process := startCommand(os.Args[1:], logger)

	scanner := bufio.NewScanner(stdout)
	scanner.Split(bufio.ScanBytes)
	currentLine := ""
	for scanner.Scan() {
		char := scanner.Text()
		fmt.Print(char)
		if char == "\n" {
			currentLine = ""
			continue
		}

		currentLine += char
		matchLine(currentLine, `HOW MANY ROLLS\? `, "5000", logger, stdin)
		matchLine(currentLine, `TRY AGAIN\? `, "N", logger, stdin)
	}

	state, err := process.Wait()
	processError(err, logger, "Error waiting for process to finish")

	fmt.Println("Exit code: ", state.ExitCode())
	os.Exit(state.ExitCode())
}

func matchLine(currentLine string, promptToMatch string, input string, logger *zap.SugaredLogger, stdin io.WriteCloser) {
	matched, err := regexp.MatchString(promptToMatch, currentLine)
	processError(err, logger, "Error matching line with regex")

	if matched {
		fmt.Print(input + "\n")
		_, err = stdin.Write([]byte(input + "\n"))
		processError(err, logger, "Error writing to stdin")
	}
}
