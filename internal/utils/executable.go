package utils

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// Terminal is what the child process finds on its standard streams.
type Terminal string

const (
	// TerminalPty gives the child a pseudo-terminal, which is what an
	// interactive program is normally run under and what it checks for when it
	// decides how to behave. It is the default.
	TerminalPty Terminal = "pty"

	// TerminalPipes gives the child ordinary pipes. A program that looks for a
	// terminal will not find one.
	TerminalPipes Terminal = "pipes"
)

// ParseTerminal turns a value from the command line into a Terminal. An empty
// string means "not specified", which callers resolve from the script.
func ParseTerminal(value string) (Terminal, error) {
	switch Terminal(value) {
	case "", TerminalPty, TerminalPipes:
		return Terminal(value), nil
	default:
		return "", fmt.Errorf("unrecognised terminal %q: expected %q or %q", value, TerminalPty, TerminalPipes)
	}
}

type Executable struct {
	Stdin  io.WriteCloser
	Stdout io.ReadCloser

	// pty is the process's end of the pseudo-terminal, held so it can be
	// closed once the child has exited. Nil when running under pipes, where
	// exec closes the pipes itself.
	pty     io.Closer
	command *exec.Cmd
	logger  Logger

	// reaped records that the child has already been waited for, because
	// waiting twice is an error and Terminate does it on the way out.
	reaped bool
}

// ExecutableOptions is everything StartExecutable needs to know about how to
// start a program. It is a struct rather than a parameter list because the
// interesting part is which fields a caller sets, not their order.
type ExecutableOptions struct {
	Path      string
	Arguments []string

	// Env is the child's environment. Nil leaves cmd.Env nil, which is exec's
	// way of saying the child inherits this process's environment.
	Env []string

	// Terminal defaults to TerminalPty when empty.
	Terminal Terminal
}

// StartExecutable starts a program and returns the two ends of its standard
// streams. Under a pty those two ends are the same file, so neither may be
// closed independently of the other.
func StartExecutable(options ExecutableOptions, logger Logger) (Executable, error) {
	executable := Executable{}

	logger.Infof("app path: %s", options.Path)
	logger.Infof("app args: \"%s\"", strings.Join(options.Arguments, "\", \""))
	if options.Env != nil {
		// Names only. A script file holds these values in plain text already,
		// but logs get pasted into bug reports and script files usually do not.
		logger.Infof("app env: %s", strings.Join(environmentNames(options.Env), ", "))
	}

	cmd := exec.Command(options.Path, options.Arguments...)
	cmd.Env = options.Env

	terminal := options.Terminal
	if terminal == "" {
		terminal = TerminalPty
	}
	logger.Infof("app terminal: %s", terminal)

	var err error
	if terminal == TerminalPipes {
		err = startWithPipes(&executable, cmd, logger)
	} else {
		err = startWithPty(&executable, cmd, logger)
	}
	if err != nil {
		return executable, err
	}

	logger.Infof("started application: %s %s", options.Path, strings.Join(options.Arguments, " "))

	executable.command = cmd
	executable.logger = logger

	return executable, nil
}

func startWithPipes(executable *Executable, cmd *exec.Cmd, logger Logger) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logger.Error("failed to capture stdout: ", err)
		return err
	}
	logger.Info("captured stdout pipe")

	cmd.Stderr = cmd.Stdout

	stdin, err := cmd.StdinPipe()
	if err != nil {
		logger.Error("failed to capture stdin: ", err)
		return err
	}
	logger.Info("captured stdin pipe")

	if err := cmd.Start(); err != nil {
		logger.Error("failed to start app: ", err)
		return err
	}

	executable.Stdin = stdin
	executable.Stdout = stdout
	return nil
}

func startWithPty(executable *Executable, cmd *exec.Cmd, logger Logger) error {
	ptmx, err := startPty(cmd)
	if err != nil {
		logger.Error("failed to start app under a pty: ", err)
		return err
	}
	logger.Info("started app under a pty")

	// One file, handed out as both ends. Closing either would close the other,
	// which is why nothing but WaitForExit closes it.
	executable.Stdin = ptmx
	executable.Stdout = ptmx
	executable.pty = ptmx
	return nil
}

func environmentNames(env []string) []string {
	names := make([]string, 0, len(env))
	for _, entry := range env {
		name, _, _ := strings.Cut(entry, "=")
		names = append(names, name)
	}
	return names
}

func (executable *Executable) WaitForExit() (int, error) {
	if executable.reaped {
		return executable.command.ProcessState.ExitCode(), nil
	}

	err := executable.command.Wait()
	executable.reaped = true
	executable.closePty()

	exitCode := executable.command.ProcessState.ExitCode()
	return exitCode, err
}

// Terminate kills a program that is not going to finish on its own, and waits
// for it. A run that gives up on a program has to take it with it: a corpus of
// a few hundred scripts that leaks one process per timeout eventually cannot
// start anything at all, and the leaked ones are usually still holding the
// terminal they were given.
func (executable *Executable) Terminate() {
	process := executable.command.Process
	if process == nil || executable.reaped {
		return
	}

	// Under a pty the child is a session leader, so whatever it started is in
	// its process group and one signal takes the lot — an interpreter that
	// spawned the program being tested is the ordinary case. Under pipes it
	// shares prescript's own group, where a group signal would kill prescript.
	if executable.pty != nil {
		if err := killGroup(process.Pid); err != nil {
			executable.logger.Debug("could not kill the process group: ", err)
		}
	}

	if err := process.Kill(); err != nil {
		executable.logger.Debug("could not kill the process: ", err)
	}

	if err := executable.command.Wait(); err != nil {
		executable.logger.Debug("process did not exit cleanly after being killed: ", err)
	}
	executable.reaped = true

	// Last, so that anything still blocked reading the pty is released only
	// once there is nothing left to read.
	executable.closePty()
}

func (executable *Executable) closePty() {
	if executable.pty == nil {
		return
	}

	if err := executable.pty.Close(); err != nil {
		executable.logger.Debug("error closing pty: ", err)
	}
	executable.pty = nil
}
