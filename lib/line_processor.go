package lib

import (
	"bufio"
	"go.uber.org/zap"
	"io"
	"time"
)

type LineProcessor struct {
	Scanner *bufio.Scanner
	logger  *zap.SugaredLogger
}

func NewLineProcessor(stdout io.ReadCloser, logger *zap.SugaredLogger) LineProcessor {
	scanner := bufio.NewScanner(stdout)
	scanner.Split(bufio.ScanBytes)
	return LineProcessor{
		Scanner: scanner,
		logger:  logger,
	}
}

type TokenResult struct {
	Finished bool
	Token    string
	Error    int
}

func (processor *LineProcessor) NextChar(timeout time.Duration) TokenResult {
	scannerChannel := make(chan bool, 0)
	go func() { scannerChannel <- processor.Scanner.Scan() }()

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

	return TokenResult{Token: processor.Scanner.Text()}
}
