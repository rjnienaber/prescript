package utils

import (
	"io"
	"os/exec"
	"strings"
)

type Executable struct {
	Stdin   io.WriteCloser
	Stdout  io.ReadCloser
	command *exec.Cmd
	logger  Logger
}

// StartExecutable runs appPath with args. A nil env leaves cmd.Env nil, which
// is exec's way of saying the child inherits this process's environment.
func StartExecutable(appPath string, args []string, env []string, logger Logger) (Executable, error) {
	executable := Executable{}

	logger.Infof("app path: %s", appPath)
	logger.Infof("app args: \"%s\"", strings.Join(args, "\", \""))
	if env != nil {
		// Names only. A script file holds these values in plain text already,
		// but logs get pasted into bug reports and script files usually do not.
		logger.Infof("app env: %s", strings.Join(environmentNames(env), ", "))
	}

	cmd := exec.Command(appPath, args...)
	cmd.Env = env
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logger.Error("failed to capture stdout: ", err)
		return executable, err
	}
	logger.Info("captured stdout pipe")

	cmd.Stderr = cmd.Stdout

	stdin, err := cmd.StdinPipe()
	if err != nil {
		logger.Error("failed to capture stdin: ", err)
		return executable, err
	}
	logger.Info("captured stdin pipe")

	err = cmd.Start()
	if err != nil {
		logger.Error("failed to start app: ", err)
		return executable, err
	}

	logger.Infof("started application: %s %s", appPath, strings.Join(args, " "))

	executable.Stdin = stdin
	executable.Stdout = stdout
	executable.command = cmd
	executable.logger = logger

	return executable, nil
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
	err := executable.command.Wait()
	exitCode := executable.command.ProcessState.ExitCode()
	return exitCode, err
}
