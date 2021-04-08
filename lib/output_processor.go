package lib

import (
	"bufio"
	"go.uber.org/zap"
	"io"
	"time"
)

type OutputProcessor struct {
	scanner *bufio.Scanner
	logger  *zap.SugaredLogger
}

func NewOutputProcessor(stdout io.ReadCloser, logger *zap.SugaredLogger) OutputProcessor {
	scanner := bufio.NewScanner(stdout)
	scanner.Split(bufio.ScanBytes)
	return OutputProcessor{
		scanner: scanner,
		logger:  logger,
	}
}

type TokenResult struct {
	Finished bool
	Token    string
	Error    int
}

func (processor *OutputProcessor) NextChar(timeout time.Duration) TokenResult {
	scannerChannel := make(chan bool, 0)
	go func() { scannerChannel <- processor.scanner.Scan() }()

	scannerResult := false
	select {
	case res := <-scannerChannel:
		scannerResult = res
	case <-time.After(timeout):
		processor.logger.Info("timed out waiting for cli to return output")
		return TokenResult{Error: CLI_ERROR}
	}

	processor.logger.Debugf("last scanner result: '%t'", scannerResult)

	if !scannerResult {
		return TokenResult{Finished: true}
	}

	return TokenResult{Token: processor.scanner.Text()}
}
