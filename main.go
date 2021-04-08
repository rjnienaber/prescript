package main

import (
	"fmt"
	"os"
	"prescript/lib"
)

func Run(params lib.RunParameters) int {
	logger := params.Logger.Sugar()
	// sync errors can be ignored: https://github.com/uber-go/zap/issues/328
	defer logger.Sync()

	command := lib.StartCommand(params.AppFilePath, params.Args, logger)
	processor := lib.NewOutputProcessor(command.Stdout, logger)
	matcher := lib.NewStepMatcher(command.Stdin, params.Steps, logger)

	timeout := params.Timeout()
	for {
		tokenResult := processor.NextChar(timeout)
		if tokenResult.Error != 0 {
			return tokenResult.Error
		}

		if tokenResult.Finished {
			break
		}

		char := tokenResult.Token
		fmt.Print(char)
		if char == "\n" {
			matcher.ResetLine()
			continue
		}

		matcher.Match(char)
	}

	command.WaitForExit()

	if matcher.MissingSteps() {
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
