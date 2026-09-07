package play

import (
	"errors"
	"fmt"
	"os"
	"strings"

	cfg "github.com/rjnienaber/prescript/internal/config"
	"github.com/rjnienaber/prescript/internal/script"
	"github.com/rjnienaber/prescript/internal/utils"
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

// Arguments supplied on the command line, after a `--` separator, take
// precedence over those recorded in the script file. This lets one script be
// replayed against a different implementation without editing it.
func getArguments(config cfg.PlayConfig, run script.Run) []string {
	if len(config.Arguments) > 0 {
		return config.Arguments
	}

	return run.Arguments
}

// reportFailure explains a failed run to whoever is watching. It writes to
// stderr rather than through the logger on purpose: the default log level is
// "none", so a failure routed through the logger is discarded and the run
// exits non-zero having said nothing at all about why. Stderr also keeps the
// report out of the executable's own output on stdout, so --quiet still
// silences the program without hiding the diagnosis.
func reportFailure(message string) {
	fmt.Fprintln(os.Stderr, message)
}

func Run(config cfg.PlayConfig, run script.Run, logger utils.Logger) int {
	executablePath, err := getExecutableFilePath(config, run)
	if err != nil {
		return utils.USER_ERROR
	}

	executable, err := utils.StartExecutable(executablePath, getArguments(config, run), run.Environment(os.Environ()), logger)
	if err != nil {
		return utils.INTERNAL_ERROR
	}

	processor := NewOutputProcessor(executable.Stdout, logger)
	matcher := NewStepMatcher(executable.Stdin, run.Steps, config.Quiet, logger)

	for {
		tokenResult := processor.NextToken(config.Timeout)
		if tokenResult.Error != nil {
			if strings.Contains(tokenResult.Error.Error(), "timed out waiting") {
				reportFailure(matcher.FailureReport(fmt.Sprintf("timed out after %s", config.Timeout)))
			} else {
				reportFailure(fmt.Sprintf("errored waiting for output from the executable: %s", tokenResult.Error))
			}

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
			matcher.EndOfLine()
			continue
		}

		err = matcher.Match(char)
		if err != nil {
			return utils.INTERNAL_ERROR
		}
	}

	exitCode, err := executable.WaitForExit()
	if matcher.MissingSteps() {
		reportFailure(matcher.FailureReport(fmt.Sprintf("the executable exited with %d before this step was reached", exitCode)))
		return utils.CLI_ERROR
	}

	if run.ExitCode == 0 && err != nil {
		logger.Error("error waiting for process to finish: ", err)
		return utils.INTERNAL_ERROR
	}

	// we rely on exit code in the script to know whether to fail on errors
	if exitCode != run.ExitCode {
		msg := fmt.Sprintf("every step matched, but the executable exited with %d and the script expects %d", exitCode, run.ExitCode)
		logger.Info(msg)
		reportFailure(msg)
		return utils.INTERNAL_ERROR
	}

	return utils.SUCCESS
}
