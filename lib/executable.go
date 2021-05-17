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

func StartExecutable(appPath string, args []string, logger *zap.SugaredLogger) Executable {
	appArgs := strings.Join(args, ",")
	cmd := exec.Command(appPath, appArgs)
	stdout, err := cmd.StdoutPipe()
	ProcessError(err, logger, "failed to capture stdout")
	logger.Debug("captured stdout pipe")

	cmd.Stderr = cmd.Stdout

	stdin, err := cmd.StdinPipe()
	ProcessError(err, logger, "failed to capture stdin")
	logger.Debug("captured stdin pipe")

	err = cmd.Start()
	ProcessError(err, logger, "failed to start app")
	logger.Debug("started application")

	return Executable{
		Stdin:   stdin,
		Stdout:  stdout,
		command: cmd,
		logger:  logger,
	}
}

func (executable *Executable) WaitForExit() {
	err := executable.command.Wait()
	ProcessError(err, executable.logger, "error waiting for process to finish")
}
