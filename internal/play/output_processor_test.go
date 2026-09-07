package play

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/rjnienaber/prescript/internal/utils"
	"github.com/stretchr/testify/assert"
)

func readAll(t *testing.T, output string, normaliseNewlines bool) string {
	t.Helper()

	processor := NewOutputProcessor(io.NopCloser(strings.NewReader(output)), normaliseNewlines, &utils.CustomLogger{})

	var read strings.Builder
	for {
		token := processor.NextToken(time.Second)
		assert.NoError(t, token.Error)
		if token.Finished {
			return read.String()
		}
		read.WriteString(token.Token)
	}
}

// A program that writes the pair itself, on a terminal that is not adding it.
// The recorded line should read the same either way, so the carriage return
// goes.
func TestCarriageReturnBeforeNewlineIsDropped(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "one\ntwo\n", readAll(t, "one\r\ntwo\r\n", true))
}

// A lone carriage return means "back to the start of the line", which is a
// thing programs do deliberately. Dropping it would silently change the output.
func TestLoneCarriageReturnIsKept(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "50%\r100%\n", readAll(t, "50%\r100%\n", true))
}

func TestCarriageReturnAtTheEndOfOutputIsKept(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "done\r", readAll(t, "done\r", true))
}

// Pipes are the compatibility route, and nothing about them introduces a
// carriage return, so nothing about them should remove one either.
func TestNormalisationIsOffForPipes(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "one\r\n", readAll(t, "one\r\n", false))
}
