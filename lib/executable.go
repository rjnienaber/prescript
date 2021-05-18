package lib

import (
	"go.uber.org/zap"
	"io"
	"os/exec"
	"strings"
)

type Executable struct {
	Stdin   io.WriteCloser
	Stdout  io.ReadCloser
	command *exec.Cmd
	logger  *zap.SugaredLogger
}

func StartExecutable(appPath string, args []string, logger *zap.SugaredLogger) (Executable, error) {
	appArgs := strings.Join(args, ",")
	executable := Executable{}

	cmd := exec.Command(appPath, appArgs)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logger.Error("failed to capture stdout: ", err)
		return executable, err
	}
	logger.Debug("captured stdout pipe")

	cmd.Stderr = cmd.Stdout

	stdin, err := cmd.StdinPipe()
	if err != nil {
		logger.Error("failed to capture stdin: ", err)
		return executable, err
	}
	logger.Debug("captured stdin pipe")

	err = cmd.Start()
	if err != nil {
		logger.Error("failed to start app: ", err)
		return executable, err
	}

	logger.Debug("started application")

	executable.Stdin = stdin
	executable.Stdout = stdout
	executable.command = cmd
	executable.logger = logger

	return executable, nil
}

func (executable *Executable) WaitForExit() error {
	err := executable.command.Wait()
	if err != nil {
		executable.logger.Error("error waiting for process to finish: ", err)
	}
	return err
}
