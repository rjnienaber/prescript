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

func startCommand(appPath string, args []string, logger *zap.SugaredLogger) (io.ReadCloser, io.WriteCloser, *os.Process) {
	appArgs := strings.Join(args, ",")
	cmd := exec.Command(appPath, appArgs)
	stdout, err := cmd.StdoutPipe()
	processError(err, logger, "Failed to capture stdout")

	cmd.Stderr = cmd.Stdout

	stdin, err := cmd.StdinPipe()
	processError(err, logger, "Failed to capture stdin")

	err = cmd.Start()
	processError(err, logger, "Failed to start app")

	return stdout, stdin, cmd.Process
}

type Step struct {
	prompt string
	input  string
}

type RunParameters struct {
	appFilePath string
	args        []string
	logger      *zap.Logger
	steps       []Step
}

func Run(params RunParameters) int {
	// sync errors can be ignored: https://github.com/uber-go/zap/issues/328
	defer params.logger.Sync()
	logger := params.logger.Sugar()

	stdout, stdin, process := startCommand(params.appFilePath, params.args, logger)

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

		for _, step := range params.steps {
			if matchLine(currentLine, step.prompt, step.input, logger, stdin) {
				currentLine = ""
				break
			}
		}
	}

	state, err := process.Wait()
	processError(err, logger, "Error waiting for process to finish")

	return state.ExitCode()
}

func main() {
	zapLogger := createLogger()
	rollStep := Step{
		prompt: `HOW MANY ROLLS\? `,
		input:  "5000",
	}

	tryAgainStep := Step{
		prompt: `TRY AGAIN\? `,
		input:  "N",
	}

	params := RunParameters{
		appFilePath: os.Args[1],
		args:        os.Args[2:],
		logger:      zapLogger,
		steps:       []Step{rollStep, tryAgainStep},
	}
	exitCode := Run(params)
	os.Exit(exitCode)
}

func matchLine(currentLine string, promptToMatch string, input string, logger *zap.SugaredLogger, stdin io.WriteCloser) bool {
	matched, err := regexp.MatchString(promptToMatch, currentLine)
	processError(err, logger, "Error matching line with regex")

	if matched {
		fmt.Print(input + "\n")
		_, err = stdin.Write([]byte(input + "\n"))
		processError(err, logger, "Error writing to stdin")
		return true
	}
	return false
}
