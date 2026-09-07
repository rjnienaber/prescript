package play

import (
	"bufio"
	"errors"
	"io"
	"os"
	"syscall"
	"time"

	"github.com/rjnienaber/prescript/internal/utils"
)

// ErrTimeout is what NextToken returns when the program said nothing for long
// enough. It is a sentinel rather than a message because the caller has to
// tell it apart from a read that actually failed, and those are two different
// things to report.
var ErrTimeout = errors.New("timed out waiting for output from the executable")

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
		// The caller kills the program: this is only the reading half, and it
		// has no business deciding the run is over.
		return utils.CapturedToken{Error: ErrTimeout}
	}

	processor.logger.Debugf("last scanner result: '%t'", scannerResult)

	if !scannerResult {
		// A scan that stops is usually the program having finished, but not
		// always, and reporting a failed read as a clean finish diagnoses it
		// as the wrong bug entirely.
		if err := processor.scanner.Err(); err != nil && !isEndOfOutput(err) {
			return utils.CapturedToken{Error: err}
		}
		return utils.CapturedToken{Finished: true}
	}

	return utils.CapturedToken{Token: processor.scanner.Text()}
}

// isEndOfOutput reports whether an error is only the program having gone away.
// Reading this end of a pty after the child exits gives EIO on Linux where a
// pipe would give EOF, and closing the pty out from under a blocked read gives
// os.ErrClosed. None of the three is a fault.
func isEndOfOutput(err error) bool {
	return errors.Is(err, io.EOF) || errors.Is(err, syscall.EIO) || errors.Is(err, os.ErrClosed)
}
