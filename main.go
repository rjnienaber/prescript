package main

import (
	"fmt"
	"go.uber.org/zap"
	"io"
	"os"
	"prescript/lib"
	"regexp"
)

func Run(params lib.RunParameters) int {
	logger := params.Logger.Sugar()
	// sync errors can be ignored: https://github.com/uber-go/zap/issues/328
	defer logger.Sync()

	command := lib.StartCommand(params.AppFilePath, params.Args, logger)

	currentLine := ""
	currentStepIndex := 0
	timeout := params.Timeout()
	for {
		tokenResult := command.LineProcessor.NextChar(timeout)
		if tokenResult.Error != 0 {
			return tokenResult.Error
		}

		if tokenResult.Finished {
			break
		}

		char := tokenResult.Token
		fmt.Print(char)
		if char == "\n" {
			currentLine = ""
			continue
		}

		currentLine += char
		if currentStepIndex < len(params.Steps) {
			step := params.Steps[currentStepIndex]
			if matchLine(currentLine, step.Line, step.Input, logger, command.Stdin) {
				currentStepIndex += 1
				currentLine = ""
			}
		}
	}

	command.WaitForExit()

	if currentStepIndex < len(params.Steps) {
		return lib.CLI_ERROR
	}

	return lib.SUCCESS
}

func main() {
	zapLogger := lib.CreateLogger()
	rollStep := lib.Step{
		Line:  `HOW MANY ROLLS\? `,
		Input: "5000",
	}

	tryAgainStep := lib.Step{
		Line:  `TRY AGAIN\? `,
		Input: "N",
	}

	params := lib.RunParameters{
		AppFilePath: os.Args[1],
		Args:        os.Args[2:],
		Logger:      zapLogger,
		Steps:       []lib.Step{rollStep, tryAgainStep},
	}
	exitCode := Run(params)
	os.Exit(exitCode)
}

func matchLine(currentLine string, currentStep string, input string, logger *zap.SugaredLogger, stdin io.WriteCloser) bool {
	matched, err := regexp.MatchString(currentStep, currentLine)
	lib.ProcessError(err, logger, "error matching line with regex")

	if matched {
		logger.Debugf("matched current line '%s' with step '%s'", currentLine, currentStep)
		if len(input) > 0 {
			fmt.Print(input + "\n")
			logger.Debugf("writing input '%s' to stdin", input)
			_, err = stdin.Write([]byte(input + "\n"))
			lib.ProcessError(err, logger, "error writing to stdin")
		}
		return true
	}
	return false
}
