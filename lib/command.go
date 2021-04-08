package lib

import (
	"go.uber.org/zap"
	"io"
	"os/exec"
	"strings"
)

type Command struct {
	Stdin   io.WriteCloser
	Stdout  io.ReadCloser
	command *exec.Cmd
	logger  *zap.SugaredLogger
}

func StartCommand(appPath string, args []string, logger *zap.SugaredLogger) Command {
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

	return Command{
		Stdin:   stdin,
		Stdout:  stdout,
		command: cmd,
		logger:  logger,
	}
}

func (cmd *Command) WaitForExit() {
	err := cmd.command.Wait()
	ProcessError(err, cmd.logger, "error waiting for process to finish")
}
