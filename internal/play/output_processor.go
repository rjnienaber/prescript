package play

import (
	"bufio"
	"errors"
	"io"
	"time"

	"github.com/rjnienaber/prescript/internal/utils"
)

type OutputProcessor struct {
	scanner *bufio.Scanner
	logger  utils.Logger
}

func NewOutputProcessor(stdout io.ReadCloser, logger utils.Logger) OutputProcessor {
	scanner := bufio.NewScanner(stdout)
	scanner.Split(bufio.ScanRunes)
	return OutputProcessor{
		scanner: scanner,
		logger:  logger,
	}
}

func (processor *OutputProcessor) NextToken(timeout time.Duration) utils.CapturedToken {
	scannerChannel := make(chan bool)
	go func() { scannerChannel <- processor.scanner.Scan() }()

	scannerResult := false
	select {
	case res := <-scannerChannel:
		scannerResult = res
	case <-time.After(timeout):
		// TODO: kill command if there is a timeout
		return utils.CapturedToken{Error: errors.New("timed out waiting for cli to return expected output")}
	}

	processor.logger.Debugf("last scanner result: '%t'", scannerResult)

	if !scannerResult {
		return utils.CapturedToken{Finished: true}
	}

	return utils.CapturedToken{Token: processor.scanner.Text()}
}
