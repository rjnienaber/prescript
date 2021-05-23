package play

import (
	"errors"
	"fmt"

	cfg "prescript/internal/config"
	"prescript/internal/script"
	"prescript/internal/utils"
)

func getExecutableFilePath(config cfg.PlayConfig, run script.Run) (string, error) {
	if config.ExecutablePath != "" {
		return config.ExecutablePath, nil
	}

	if run.Executable != "" {
		return run.Executable, nil
	}

	return "", errors.New("could not find executable path in argument or script file")
}

func Run(config cfg.PlayConfig, run script.Run, logger utils.Logger) int {
	executablePath, err := getExecutableFilePath(config, run)
	if err != nil {
		return utils.USER_ERROR
	}

	executable, err := utils.StartExecutable(executablePath, run.Arguments, logger)
	if err != nil {
		return utils.INTERNAL_ERROR
	}

	processor := NewOutputProcessor(executable.Stdout, logger)
	matcher := NewStepMatcher(executable.Stdin, run.Steps, config.Quiet, logger)

	for {
		tokenResult := processor.NextToken(config.Timeout)
		if tokenResult.Error != nil {
			return utils.CLI_ERROR
		}

		if tokenResult.Finished {
			break
		}

		char := tokenResult.Token
		if !config.Quiet {
			fmt.Print(char)
		}

		if char == "\n" {
			matcher.ResetLine()
			continue
		}

		err = matcher.Match(char)
		if err != nil {
			return utils.INTERNAL_ERROR
		}
	}

	exitCode, err := executable.WaitForExit()
	if matcher.MissingSteps() {
		logger.Debug("executable finished but there are missing steps")
		return utils.CLI_ERROR
	}

	if run.ExitCode == 0 && err != nil {
		logger.Error("error waiting for process to finish: ", err)
		return utils.INTERNAL_ERROR
	}

	// we rely on exit code in the script to know whether to fail on errors
	if exitCode != run.ExitCode {
		msg := fmt.Sprintf("exit code from script (%d) did not match exit code from executable (%d)", run.ExitCode, exitCode)
		logger.Info(msg)
		return utils.INTERNAL_ERROR
	}

	return utils.SUCCESS
}
