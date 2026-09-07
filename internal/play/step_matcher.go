package play

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/rjnienaber/prescript/internal/script"
	"github.com/rjnienaber/prescript/internal/utils"
)

// maxReportedLines caps how much received output a failure report quotes back.
// Enough to show where a run went off course, few enough to paste into a bug
// report without burying the interesting line.
const maxReportedLines = 8

type StepMatcher struct {
	currentLine      string
	linesSinceMatch  []string
	currentStepIndex int
	logger           utils.Logger
	stdin            io.WriteCloser
	steps            []script.Step
	quiet            bool
}

func NewStepMatcher(stdin io.WriteCloser, steps []script.Step, quiet bool, logger utils.Logger) StepMatcher {
	return StepMatcher{
		logger: logger,
		stdin:  stdin,
		steps:  steps,
		quiet:  quiet,
	}
}

func (matcher *StepMatcher) MissingSteps() bool {
	return matcher.currentStepIndex < len(matcher.steps)
}

func (matcher *StepMatcher) ResetLine() {
	matcher.currentLine = ""
}

// EndOfLine finishes the line being matched and keeps it as context for a
// failure report. Only lines since the last successful match are kept:
// everything before that is output the script has already accounted for.
func (matcher *StepMatcher) EndOfLine() {
	matcher.linesSinceMatch = append(matcher.linesSinceMatch, matcher.currentLine)
	if len(matcher.linesSinceMatch) > maxReportedLines {
		matcher.linesSinceMatch = matcher.linesSinceMatch[len(matcher.linesSinceMatch)-maxReportedLines:]
	}
	matcher.ResetLine()
}

func (matcher *StepMatcher) NextExpectedLine() string {
	if len(matcher.steps) == matcher.currentStepIndex {
		return ""
	} else {
		return matcher.steps[matcher.currentStepIndex].Line
	}
}

// receivedLine is the best candidate for what the current step saw instead of
// what it wanted. A prompt arrives without a trailing newline, so the line
// still being matched is usually it; otherwise the last completed line is.
func (matcher *StepMatcher) receivedLine() string {
	if matcher.currentLine != "" {
		return matcher.currentLine
	}
	if len(matcher.linesSinceMatch) > 0 {
		return matcher.linesSinceMatch[len(matcher.linesSinceMatch)-1]
	}
	return ""
}

// FailureReport explains why a run stopped short, in a form meant to be read
// by a person or pasted into a bug report. reason says what ended the run, for
// example "timed out after 30s".
//
// Lines are quoted because trailing whitespace is load-bearing: prompts
// routinely end in a space, and an unquoted report makes a step that differs
// only in that respect look identical to the one it failed to match.
func (matcher *StepMatcher) FailureReport(reason string) string {
	if !matcher.MissingSteps() {
		return fmt.Sprintf("all %d steps matched, but the run failed: %s", len(matcher.steps), reason)
	}

	step := matcher.steps[matcher.currentStepIndex]
	expected, received := matcher.NextExpectedLine(), matcher.receivedLine()

	var report strings.Builder
	fmt.Fprintf(&report, "step %d of %d did not match (%s)\n\n",
		matcher.currentStepIndex+1, len(matcher.steps), reason)
	switch {
	case step.Redacted:
		fmt.Fprintf(&report, "  expected  %q  (with redactions applied)\n", expected)
	case step.IsRegex:
		fmt.Fprintf(&report, "  expected  %q  (regular expression)\n", expected)
	default:
		fmt.Fprintf(&report, "  expected  %q\n", expected)
	}
	fmt.Fprintf(&report, "  received  %q\n", received)
	// A character-level pointer only means something when the two are meant to
	// be equal; against a pattern the first differing byte is noise.
	if !step.UsesPattern() {
		report.WriteString(differenceMarker(expected, received, len(reportLabelIndent)))
	}

	// The preceding lines are what makes a mismatch diagnosable when the
	// received line is not the interesting one, e.g. the executable printed an
	// error and exited before ever reaching the prompt.
	if context := matcher.contextLines(); len(context) > 0 {
		report.WriteString("\n  output since the last matched step:\n")
		for _, line := range context {
			fmt.Fprintf(&report, "    %s\n", line)
		}
	}

	if remaining := len(matcher.steps) - matcher.currentStepIndex - 1; remaining > 0 {
		fmt.Fprintf(&report, "\n%s never reached\n", pluralSteps(remaining))
	}

	return report.String()
}

// contextLines are the completed lines leading up to the failure, excluding the
// one already shown as `received`.
func (matcher *StepMatcher) contextLines() []string {
	lines := matcher.linesSinceMatch
	if matcher.currentLine == "" && len(lines) > 0 {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// reportLabelIndent is the prefix in front of the quoted expected/received
// lines. differenceMarker needs its width to line a caret up underneath.
const reportLabelIndent = "  received  "

// differenceMarker points at the first character where the received line
// diverged from the expected one. Spotting that by eye in two quoted strings
// is exactly the tedium this tool exists to remove, and it is worst in the
// cases that matter most: a changed letter mid-word, or a trailing space.
func differenceMarker(expected, received string, indent int) string {
	// With nothing received there is nothing to point at, and the empty quotes
	// above already say it plainly.
	if expected == received || received == "" {
		return ""
	}

	prefix := commonPrefix(expected, received)
	label := "first difference"
	if prefix == received {
		label = "received output ends here"
	}

	// The caret indexes the quoted rendering, so measure the prefix as it will
	// appear once escaped. Quote adds a pair of quotes; only the opening one
	// sits between the indent and the first differing character.
	column := indent + len(strconv.Quote(prefix)) - 1
	return fmt.Sprintf("%s^ %s\n", strings.Repeat(" ", column), label)
}

func commonPrefix(a, b string) string {
	offset := 0
	for _, char := range a {
		width := utf8.RuneLen(char)
		if offset+width > len(b) || a[offset:offset+width] != b[offset:offset+width] {
			break
		}
		offset += width
	}
	return a[:offset]
}

func pluralSteps(count int) string {
	if count == 1 {
		return "1 later step was"
	}
	return fmt.Sprintf("%d later steps were", count)
}

func (matcher *StepMatcher) Match(char string) error {
	matcher.currentLine += char
	if matcher.currentStepIndex < len(matcher.steps) {
		step := matcher.steps[matcher.currentStepIndex]
		matched, err := matcher.matchLine(step)
		if err != nil {
			return err
		}
		if matched {
			matcher.currentStepIndex += 1
			matcher.linesSinceMatch = nil
			matcher.ResetLine()
		}
	}
	return nil
}

func (matcher *StepMatcher) matchLine(step script.Step) (bool, error) {
	var matched bool
	if step.UsesPattern() {
		matched = step.LineRegex.MatchString(matcher.currentLine)
	} else {
		matched = matcher.currentLine == step.Line
	}

	if matched {
		matcher.logger.Debugf("matched current line '%s' with step '%s'", matcher.currentLine, step.Line)
		if len(step.Input) > 0 {
			if !matcher.quiet {
				fmt.Print(step.Input + "\n")
			}

			matcher.logger.Debugf("writing input '%s' to stdin", step.Input)
			_, err := matcher.stdin.Write([]byte(step.Input + "\n"))
			if err != nil {
				matcher.logger.Debug("error writing user input to stdin: ", err)
				return false, err
			}
		}
		return true, nil
	}
	return false, nil
}
