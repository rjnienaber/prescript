package lib

import (
	"errors"
	"fmt"
)

func getExecutableFilePath(config Config, run Run) (string, error) {
	if config.Play.Executable != "" {
		return config.Play.Executable, nil
	}

	if run.Executable != "" {
		return run.Executable, nil
	}

	return "", errors.New("could not find executable path in argument or script file")
}

func RunPlay(config Config, run Run) int {
	executablePath, err := getExecutableFilePath(config, run)
	if err != nil {
		return USER_ERROR
	}

	executable := StartExecutable(executablePath, run.Arguments, config.Logger)
	processor := NewOutputProcessor(executable.Stdout, config.Logger)
	matcher := NewStepMatcher(executable.Stdin, run.Steps, config.Logger)

	for {
		tokenResult := processor.NextChar(config.Play.Timeout)
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

	executable.WaitForExit()

	if matcher.MissingSteps() {
		return CLI_ERROR
	}

	return SUCCESS
}
