package lib

import (
	"go.uber.org/zap"
	"io"
	"os/exec"
	"strings"
)

type Command struct {
	Stdin         io.WriteCloser
	command       *exec.Cmd
	LineProcessor LineProcessor
	Logger        *zap.SugaredLogger
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

	lineProcessor := NewLineProcessor(stdout, logger)

	return Command{
		Stdin:         stdin,
		command:       cmd,
		LineProcessor: lineProcessor,
		Logger:        logger,
	}
}

func (cmd *Command) WaitForExit() {
	err := cmd.command.Wait()
	ProcessError(err, cmd.Logger, "error waiting for process to finish")
}
