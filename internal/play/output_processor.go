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

	// normaliseNewlines drops the carriage return of a CRLF pair. See
	// NextToken.
	normaliseNewlines bool

	// pending holds a token read ahead of its turn, which happens only when a
	// carriage return turned out not to be part of a CRLF pair.
	pending *utils.CapturedToken
}

// NewOutputProcessor reads a program's output one character at a time.
//
// normaliseNewlines belongs with a pty: a terminal's line discipline is what
// turns "\n" into "\r\n", and a program that has detected a terminal may write
// the pair itself. prescript clears that translation on the pty it opens, so
// this is a backstop for the program that does it on its own — without it, a
// step would have to expect a carriage return that a recording made through a
// pipe never contained.
func NewOutputProcessor(stdout io.ReadCloser, normaliseNewlines bool, logger utils.Logger) OutputProcessor {
	scanner := bufio.NewScanner(stdout)
	scanner.Split(bufio.ScanRunes)
	return OutputProcessor{
		scanner:           scanner,
		normaliseNewlines: normaliseNewlines,
		logger:            logger,
	}
}

func (processor *OutputProcessor) NextToken(timeout time.Duration) utils.CapturedToken {
	token := processor.read(timeout)
	if !processor.normaliseNewlines || token.Token != "\r" {
		return token
	}

	// A lone carriage return is left alone: it means "back to the start of the
	// line", which is a thing programs do deliberately and not something to
	// silently discard. Only the pair collapses.
	next := processor.read(timeout)
	if next.Token == "\n" {
		return next
	}

	processor.pending = &next
	return token
}

func (processor *OutputProcessor) read(timeout time.Duration) utils.CapturedToken {
	if processor.pending != nil {
		token := *processor.pending
		processor.pending = nil
		return token
	}

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
