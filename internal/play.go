package internal

import (
	"errors"
	"fmt"

	cfg "prescript/internal/config"
)

func getExecutableFilePath(config cfg.Config, run Run) (string, error) {
	if config.Play.ExecutablePath != "" {
		return config.Play.ExecutablePath, nil
	}

	if run.Executable != "" {
		return run.Executable, nil
	}

	return "", errors.New("could not find executable path in argument or script file")
}

func RunPlay(config cfg.Config, run Run) int {
	executablePath, err := getExecutableFilePath(config, run)
	if err != nil {
		return USER_ERROR
	}

	executable, err := StartExecutable(executablePath, run.Arguments, config.Logger)
	if err != nil {
		return INTERNAL_ERROR
	}

	processor := NewOutputProcessor(executable.Stdout, config.Logger)
	matcher := NewStepMatcher(executable.Stdin, run.Steps, config)

	for {
		tokenResult := processor.NextChar(config.Play.Timeout)
		if tokenResult.Error != 0 {
			return tokenResult.Error
		}

		if tokenResult.Finished {
			break
		}

		char := tokenResult.Token
		if !config.Play.Quiet {
			fmt.Print(char)
		}

		if char == "\n" {
			matcher.ResetLine()
			continue
		}

		err := matcher.Match(char)
		if err != nil {
			return INTERNAL_ERROR
		}
	}

	exitCode, err := executable.WaitForExit()
	if matcher.MissingSteps() {
		logger.Debug("executable finished but there are missing steps")
		return CLI_ERROR
	}

	// we rely on exit code in the script to know whether to fail on errors
	if run.ExitCode == 0 && err != nil {
		executable.logger.Error("error waiting for process to finish: ", err)
		return INTERNAL_ERROR
	}

	// we rely on exit code in the script to know whether to fail on errors
	if exitCode != run.ExitCode {
		msg := fmt.Sprintf("exit code from script (%d) did not match exit code from executable (%d)", run.ExitCode, exitCode)
		logger.Info(msg)
		return INTERNAL_ERROR
	}

	return SUCCESS
}
