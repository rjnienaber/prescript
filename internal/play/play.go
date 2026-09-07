package play

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

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
//
// A runner's arguments are not part of that contest and always come first:
// they name the interpreter's own flags, while everything after them names the
// program to feed it.
func getArguments(config cfg.PlayConfig, run script.Run) []string {
	arguments := run.Arguments
	if len(config.Arguments) > 0 {
		arguments = config.Arguments
	}

	if len(run.RunnerArguments) == 0 {
		return arguments
	}

	return append(append([]string{}, run.RunnerArguments...), arguments...)
}

// getTerminal decides what the program is given for its standard streams.
// --terminal is an override for trying a script the other way round without
// editing it, so it wins over what the script or runner declared.
func getTerminal(config cfg.PlayConfig, run script.Run) (utils.Terminal, error) {
	if config.Terminal != "" {
		return utils.ParseTerminal(config.Terminal)
	}

	// A value from a file has already been through the schema's enum, so this
	// only re-checks what the schema has said.
	return utils.ParseTerminal(run.Terminal)
}

// tapeVariable is how a shim is told where the tape is. One variable for every
// language, because a corpus run chooses a tape once and plays it against
// every port.
const tapeVariable = "PRESCRIPT_TAPE"

// childEnvironment is the environment the executable is started with, plus the
// tape if one was named.
//
// The tape is appended rather than declared, and that is the point: a run with
// no env of its own inherits prescript's whole environment, and adding a
// variable to the declared set would quietly turn that into an environment of
// exactly one variable. exec keeps the last assignment to a name, so appending
// also means --tape wins over a runner that named one, the same way --terminal
// does.
//
// The path is made absolute because the value outlives this process's idea of
// where it is: it is read by a shim, inside an interpreter, started by a
// runner that may have been written anywhere.
func childEnvironment(config cfg.PlayConfig, run script.Run) ([]string, error) {
	environment := run.Environment(os.Environ())
	if config.Tape == "" {
		return environment, nil
	}

	tape, err := filepath.Abs(config.Tape)
	if err != nil {
		return nil, err
	}

	// Checked before the run rather than discovered during it. A missing tape
	// is a mistake in the command line, and a program that finds out halfway
	// through has already printed half a transcript that means nothing.
	if _, err := os.Stat(tape); err != nil {
		return nil, fmt.Errorf("could not read the tape: %w", err)
	}

	if environment == nil {
		environment = os.Environ()
	}
	return append(environment, tapeVariable+"="+tape), nil
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

	terminal, err := getTerminal(config, run)
	if err != nil {
		reportFailure(err.Error())
		return utils.USER_ERROR
	}

	environment, err := childEnvironment(config, run)
	if err != nil {
		reportFailure(err.Error())
		return utils.USER_ERROR
	}

	executable, err := utils.StartExecutable(utils.ExecutableOptions{
		Path:      executablePath,
		Arguments: getArguments(config, run),
		Env:       environment,
		Terminal:  terminal,
	}, logger)
	if err != nil {
		return utils.INTERNAL_ERROR
	}

	processor := NewOutputProcessor(executable.Stdout, terminal != utils.TerminalPipes, logger)
	matcher := NewStepMatcher(executable.Stdin, run.Steps, config.Quiet, logger)

	for {
		timeout := matcher.Timeout(config.Timeout)
		tokenResult := processor.NextToken(timeout)
		if tokenResult.Error != nil {
			return reportInterrupted(&executable, &matcher, timeout, tokenResult.Error)
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
		reportFailure(matcher.FailureReport(ExitedEarly, fmt.Sprintf("the executable exited with %d", exitCode)))
		return utils.CLI_ERROR
	}

	// A non-zero exit arrives as an ExitError, and that is the program's answer
	// rather than a fault: the script says which code it expects, and the
	// comparison below is what decides. Anything else from Wait means the run
	// could not be completed at all.
	var exitError *exec.ExitError
	if err != nil && !errors.As(err, &exitError) {
		logger.Error("error waiting for process to finish: ", err)
		return utils.INTERNAL_ERROR
	}

	// we rely on exit code in the script to know whether to fail on errors
	if exitCode != run.ExitCode {
		msg := matcher.FailureReport(WrongExitCode,
			fmt.Sprintf("the executable exited with %d and the script expects %d", exitCode, run.ExitCode))
		logger.Info(msg)
		reportFailure(msg)
		return utils.INTERNAL_ERROR
	}

	return utils.SUCCESS
}

// reportInterrupted deals with a run that stopped while the program was still
// running, which is either a timeout or a read that failed outright.
//
// The program is killed first, before anything is written. Whatever it was
// doing, it is not going to be asked for anything else, and a run that walks
// away from it leaves it holding a terminal and, in a corpus run, one more
// process than the last time.
//
// Which timeout it is matters as much as that there was one. A program that is
// still printing something the script does not expect is a divergence to look
// at; a program that matched every step and then would not exit is waiting for
// something nobody is going to send. Those are different bugs and they are
// named differently.
func reportInterrupted(executable *utils.Executable, matcher *StepMatcher, timeout time.Duration, err error) int {
	executable.Terminate()

	switch {
	case !errors.Is(err, ErrTimeout):
		reportFailure(matcher.FailureReport(ReadFailed, fmt.Sprintf("could not read from the executable: %s", err)))
	case matcher.MissingSteps():
		reportFailure(matcher.FailureReport(NoMatch, fmt.Sprintf("nothing matched it within %s", timeout)))
	default:
		reportFailure(matcher.FailureReport(Hung, fmt.Sprintf("the executable had not exited after %s", timeout)))
	}

	return utils.CLI_ERROR
}
