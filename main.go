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

const INTERNAL_ERROR = 2

func processError(err error, logger *zap.SugaredLogger, message string) {
	if err != nil {
		logger.Error(message+": ", err)
		os.Exit(INTERNAL_ERROR)
	}
}

func createLogger() *zap.Logger {
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Println("Could not create logger", err.Error())
		os.Exit(INTERNAL_ERROR)
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
	line  string
	input string
}

// TODO: embed struct for <script.json> with RunParameters
// https://stackoverflow.com/a/40510391/9539
type RunParameters struct {
	appFilePath           string
	args                  []string
	timeoutInMilliseconds int
	logger                *zap.Logger
	steps                 []Step
	exitCode              int
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
		// TODO: verify output steps
		char := scanner.Text()
		fmt.Print(char)
		if char == "\n" {
			// TODO: should match lines with steps
			currentLine = ""
			continue
		}

		currentLine += char

		for _, step := range params.steps {
			// TODO: should not loop over steps that have already matched
			if matchLine(currentLine, step.line, step.input, logger, stdin) {
				currentLine = ""
				break
			}
		}
	}

	state, err := process.Wait()
	processError(err, logger, "Error waiting for process to finish")

	exitCode := state.ExitCode()
	return exitCode
}

func main() {
	zapLogger := createLogger()
	rollStep := Step{
		line:  `HOW MANY ROLLS\? `,
		input: "5000",
	}

	tryAgainStep := Step{
		line:  `TRY AGAIN\? `,
		input: "N",
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
