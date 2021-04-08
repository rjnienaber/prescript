package main

import (
	"fmt"
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
			if matchLine(command, step, currentLine) {
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

func matchLine(command lib.Command, step lib.Step, currentLine string) bool {
	matched, err := regexp.MatchString(step.Line, currentLine)
	lib.ProcessError(err, command.Logger, "error matching line with regex")

	if matched {
		command.Logger.Debugf("matched current line '%s' with step '%s'", currentLine, step.Line)
		if len(step.Input) > 0 {
			fmt.Print(step.Input + "\n")
			command.Logger.Debugf("writing input '%s' to stdin", step.Input)
			_, err = command.Stdin.Write([]byte(step.Input + "\n"))
			lib.ProcessError(err, command.Logger, "error writing to stdin")
		}
		return true
	}
	return false
}
