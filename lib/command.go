package lib

import (
	"bufio"
	"go.uber.org/zap"
	"io"
	"os"
	"os/exec"
	"strings"
)

type Command struct {
	Stdout  io.ReadCloser
	Stdin   io.WriteCloser
	Process *os.Process
	Scanner *bufio.Scanner
	Logger  *zap.SugaredLogger
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

	scanner := bufio.NewScanner(stdout)
	scanner.Split(bufio.ScanBytes)

	return Command{
		Stdout:  stdout,
		Stdin:   stdin,
		Process: cmd.Process,
		Scanner: scanner,
		Logger:  logger,
	}
}
