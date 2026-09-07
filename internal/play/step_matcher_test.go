package play

import (
	"regexp"
	"strings"
	"testing"

	"github.com/rjnienaber/prescript/internal/script"
	"github.com/rjnienaber/prescript/internal/utils"
	"github.com/stretchr/testify/assert"
)

type discardStdin struct{}

func (discardStdin) Write(bytes []byte) (int, error) { return len(bytes), nil }
func (discardStdin) Close() error                    { return nil }

func newTestMatcher(steps []script.Step) StepMatcher {
	logger := utils.CustomLogger{}
	return NewStepMatcher(discardStdin{}, steps, true, &logger)
}

// feed drives the matcher the way Run does: one character at a time, with
// newlines ending the current line rather than being matched against a step.
func feed(t *testing.T, matcher *StepMatcher, output string) {
	t.Helper()
	for _, char := range output {
		if char == '\n' {
			matcher.EndOfLine()
			continue
		}
		assert.NoError(t, matcher.Match(string(char)))
	}
}

// The caret has to land under the first differing character of the *quoted*
// received line, which is why it is asserted by position rather than by eye.
func caretColumn(t *testing.T, report string) int {
	t.Helper()
	for _, line := range strings.Split(report, "\n") {
		if index := strings.Index(line, "^"); index >= 0 && strings.TrimSpace(line)[0] == '^' {
			return index
		}
	}
	return -1
}

func TestFailureReportShowsWhatWasReceived(t *testing.T) {
	t.Parallel()
	matcher := newTestMatcher([]script.Step{{Line: "HOW MANY ROLES? "}})

	feed(t, &matcher, "HOW MANY ROLLS? ")
	report := matcher.FailureReport("timed out after 5s")

	assert.Contains(t, report, "step 1 of 1 did not match (timed out after 5s)")
	assert.Contains(t, report, `expected  "HOW MANY ROLES? "`)
	assert.Contains(t, report, `received  "HOW MANY ROLLS? "`)

	// "HOW MANY ROL" is common, so the caret belongs under the second L: the
	// label indent, the opening quote, then twelve matching characters.
	assert.Equal(t, len(reportLabelIndent)+1+len("HOW MANY ROL"), caretColumn(t, report))
	assert.Contains(t, report, "first difference")
}

func TestFailureReportQuotesTrailingWhitespace(t *testing.T) {
	t.Parallel()
	matcher := newTestMatcher([]script.Step{{Line: "Name: "}})

	feed(t, &matcher, "Name:")
	report := matcher.FailureReport("timed out after 5s")

	// Unquoted, these two lines would look identical, which is the whole
	// reason for quoting them.
	assert.Contains(t, report, `expected  "Name: "`)
	assert.Contains(t, report, `received  "Name:"`)
	assert.Contains(t, report, "received output ends here")
}

func TestFailureReportOmitsMarkerWhenNothingWasReceived(t *testing.T) {
	t.Parallel()
	matcher := newTestMatcher([]script.Step{{Line: "THANKS FOR PLAYING"}})

	report := matcher.FailureReport("the executable exited with 0 before this step was reached")

	assert.Contains(t, report, `received  ""`)
	assert.NotContains(t, report, "^")
}

func TestFailureReportOmitsMarkerForRegexSteps(t *testing.T) {
	t.Parallel()
	pattern := regexp.MustCompile(`^Rolled \d+$`)
	matcher := newTestMatcher([]script.Step{{Line: pattern.String(), IsRegex: true, LineRegex: *pattern}})

	feed(t, &matcher, "Rolled twelve")
	report := matcher.FailureReport("timed out after 5s")

	assert.Contains(t, report, "(regular expression)")
	// Pointing at the first byte where output differs from a pattern would be
	// meaningless.
	assert.NotContains(t, report, "first difference")
}

func TestFailureReportIncludesPrecedingOutput(t *testing.T) {
	t.Parallel()
	matcher := newTestMatcher([]script.Step{{Line: "Ready? "}})

	feed(t, &matcher, "loading data\nchecking licence\nlicence expired\n")
	report := matcher.FailureReport("timed out after 5s")

	assert.Contains(t, report, "output since the last matched step:")
	assert.Contains(t, report, "loading data")
	assert.Contains(t, report, "checking licence")
	// The last completed line doubles as `received`, so it is not repeated.
	assert.Equal(t, 1, strings.Count(report, "licence expired"))
}

func TestContextOnlyCoversOutputSinceTheLastMatch(t *testing.T) {
	t.Parallel()
	matcher := newTestMatcher([]script.Step{
		{Line: "First number: ", Input: "1"},
		{Line: "Second number: "},
	})

	feed(t, &matcher, "welcome\nFirst number: ")
	feed(t, &matcher, "\nthinking\nSecond numbr: ")
	report := matcher.FailureReport("timed out after 5s")

	assert.Contains(t, report, "step 2 of 2 did not match")
	assert.Contains(t, report, "thinking")
	// Output the script already accounted for is noise in a bug report.
	assert.NotContains(t, report, "welcome")
}

func TestContextIsCapped(t *testing.T) {
	t.Parallel()
	matcher := newTestMatcher([]script.Step{{Line: "Ready? "}})

	var output strings.Builder
	for line := 0; line < maxReportedLines*3; line++ {
		output.WriteString("line\n")
	}
	feed(t, &matcher, output.String())
	report := matcher.FailureReport("timed out after 5s")

	assert.Equal(t, maxReportedLines, strings.Count(report, "line"))
}

func TestFailureReportCountsStepsNeverReached(t *testing.T) {
	t.Parallel()
	steps := []script.Step{{Line: "one"}, {Line: "two"}, {Line: "three"}}

	reportFor := func(steps []script.Step) string {
		matcher := newTestMatcher(steps)
		return matcher.FailureReport("timed out")
	}

	assert.Contains(t, reportFor(steps), "2 later steps were never reached")
	assert.Contains(t, reportFor(steps[:2]), "1 later step was never reached")
	assert.NotContains(t, reportFor(steps[:1]), "never reached")
}

func TestFailureReportWhenEveryStepMatched(t *testing.T) {
	t.Parallel()
	matcher := newTestMatcher([]script.Step{{Line: "Ready? "}})

	feed(t, &matcher, "Ready? ")
	report := matcher.FailureReport("timed out after 5s")

	assert.False(t, matcher.MissingSteps())
	assert.Equal(t, "all 1 steps matched, but the run failed: timed out after 5s", report)
}
