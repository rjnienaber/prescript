package play

import (
	"fmt"
	"io"
	"time"

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

// Timeout is how long to wait for the next character to arrive. A step that
// knows it is slow — a program that computes for a minute before printing its
// next prompt — says so for itself, so the rest of the script does not have to
// be given the slowest step's patience as well.
//
// The wait is per character rather than per step, which is what the run-wide
// timeout has always meant: it bounds how long the program may go on saying
// nothing, not how long the step may take in total.
func (matcher *StepMatcher) Timeout(runTimeout time.Duration) time.Duration {
	if !matcher.MissingSteps() {
		return runTimeout
	}

	if step := matcher.steps[matcher.currentStepIndex]; step.TimeoutDuration > 0 {
		return step.TimeoutDuration
	}
	return runTimeout
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

// Divergence records where this run stopped agreeing with its transcript.
// mode names which of the ways it went wrong, and detail says what was
// observed, for example "the executable exited with 1".
func (matcher *StepMatcher) Divergence(mode FailureMode, detail string) Divergence {
	divergence := Divergence{
		Mode:      mode,
		Detail:    detail,
		StepIndex: matcher.currentStepIndex,
		StepCount: len(matcher.steps),
	}

	if !matcher.MissingSteps() {
		return divergence
	}

	step := matcher.steps[matcher.currentStepIndex]
	divergence.Expected = step.Line
	divergence.Received = matcher.receivedLine()
	divergence.Redacted = step.Redacted
	divergence.Context = matcher.contextLines()
	return divergence
}

// FailureReport is the divergence written out. Kept as one call because the
// overwhelmingly common case -- a single run, reported to stderr and read by a
// person -- has no use for the value in between.
func (matcher *StepMatcher) FailureReport(mode FailureMode, detail string) string {
	return matcher.Divergence(mode, detail).Report()
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
	if step.Redacted {
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
