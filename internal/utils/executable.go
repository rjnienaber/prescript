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

func StartExecutable(appPath string, args []string, logger Logger) (Executable, error) {
	executable := Executable{}

	logger.Infof("app path: %s", appPath)
	logger.Infof("app args: \"%s\"", strings.Join(args, "\", \""))

	cmd := exec.Command(appPath, args...)
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
	executable.command.ProcessState.ExitCode()

	return executable, nil
}

func (executable *Executable) WaitForExit() (int, error) {
	err := executable.command.Wait()
	exitCode := executable.command.ProcessState.ExitCode()
	return exitCode, err
}
