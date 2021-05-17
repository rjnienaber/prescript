package lib

import "fmt"

func RunPlay(params RunParameters) int {
	logger := params.Logger.Sugar()
	// sync errors can be ignored: https://github.com/uber-go/zap/issues/328
	defer logger.Sync()

	command := StartCommand(params.AppFilePath, params.Args, logger)
	processor := NewOutputProcessor(command.Stdout, logger)
	matcher := NewStepMatcher(command.Stdin, params.Steps, logger)

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
		return CLI_ERROR
	}

	return SUCCESS
}
