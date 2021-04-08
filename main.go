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
	"time"
)

const SUCCESS = 0
const CLI_ERROR = 1
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
		fmt.Println("could not create logger", err.Error())
		os.Exit(INTERNAL_ERROR)
	}
	return logger
}

func startCommand(appPath string, args []string, logger *zap.SugaredLogger) (io.ReadCloser, io.WriteCloser, *os.Process) {
	appArgs := strings.Join(args, ",")
	cmd := exec.Command(appPath, appArgs)
	stdout, err := cmd.StdoutPipe()
	processError(err, logger, "failed to capture stdout")
	logger.Debug("captured stdout pipe")

	cmd.Stderr = cmd.Stdout

	stdin, err := cmd.StdinPipe()
	processError(err, logger, "failed to capture stdin")
	logger.Debug("captured stdin pipe")

	err = cmd.Start()
	processError(err, logger, "failed to start app")
	logger.Debug("started application")

	return stdout, stdin, cmd.Process
}

type Step struct {
	line  string
	input string
}

// TODO: merge struct for <script.json> with RunParameters
// https://stackoverflow.com/a/40510391/9539
type RunParameters struct {
	appFilePath           string
	args                  []string
	timeoutInMilliseconds int64
	logger                *zap.Logger
	steps                 []Step
	exitCode              int
}

func (params *RunParameters) timeout() time.Duration {
	if params.timeoutInMilliseconds != 0 {
		return time.Duration(params.timeoutInMilliseconds) * time.Millisecond
	}
	return 30 * time.Millisecond
}

func Run(params RunParameters) int {
	// sync errors can be ignored: https://github.com/uber-go/zap/issues/328
	defer params.logger.Sync()
	logger := params.logger.Sugar()

	stdout, stdin, process := startCommand(params.appFilePath, params.args, logger)

	scanner := bufio.NewScanner(stdout)
	scanner.Split(bufio.ScanBytes)
	currentLine := ""
	currentStepIndex := 0
	for {
		scannerChannel := make(chan bool, 0)
		go func() { scannerChannel <- scanner.Scan() }()

		scannerResult := false
		select {
		case res := <-scannerChannel:
			scannerResult = res
		case <-time.After(params.timeout()):
			logger.Info("timed out waiting for cli to return output")
			return CLI_ERROR
		}

		logger.Debugf("last scanner result: '%t'", scannerResult)

		if !scannerResult {
			break
		}

		char := scanner.Text()
		fmt.Print(char)
		if char == "\n" {
			currentLine = ""
			continue
		}

		currentLine += char
		if currentStepIndex < len(params.steps) {
			step := params.steps[currentStepIndex]
			if matchLine(currentLine, step.line, step.input, logger, stdin) {
				currentStepIndex += 1
				currentLine = ""
			}
		}
	}

	_, err := process.Wait()
	processError(err, logger, "error waiting for process to finish")

	if currentStepIndex < len(params.steps) {
		return CLI_ERROR
	}

	return SUCCESS
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

func matchLine(currentLine string, currentStep string, input string, logger *zap.SugaredLogger, stdin io.WriteCloser) bool {
	matched, err := regexp.MatchString(currentStep, currentLine)
	processError(err, logger, "error matching line with regex")

	if matched {
		logger.Debugf("matched current line '%s' with step '%s'", currentLine, currentStep)
		if len(input) > 0 {
			fmt.Print(input + "\n")
			logger.Debugf("writing input '%s' to stdin", input)
			_, err = stdin.Write([]byte(input + "\n"))
			processError(err, logger, "error writing to stdin")
		}
		return true
	}
	return false
}
