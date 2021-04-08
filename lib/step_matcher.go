package lib

import (
	"fmt"
	"go.uber.org/zap"
	"io"
	"regexp"
)

type StepMatcher struct {
	currentLine      string
	currentStepIndex int
	logger           *zap.SugaredLogger
	stdin            io.WriteCloser
	steps            []Step
}

func NewStepMatcher(stdin io.WriteCloser, steps []Step, logger *zap.SugaredLogger) StepMatcher {
	return StepMatcher{
		logger: logger,
		stdin:  stdin,
		steps:  steps,
	}
}

func (matcher *StepMatcher) MissingSteps() bool {
	return matcher.currentStepIndex < len(matcher.steps)
}

func (matcher *StepMatcher) ResetLine() {
	matcher.currentLine = ""
}

func (matcher *StepMatcher) Match(char string) {
	matcher.currentLine += char
	if matcher.currentStepIndex < len(matcher.steps) {
		step := matcher.steps[matcher.currentStepIndex]
		if matcher.matchLine(step) {
			matcher.currentStepIndex += 1
			matcher.ResetLine()
		}
	}
}

func (matcher *StepMatcher) matchLine(step Step) bool {
	matched := false
	if step.IsRegex {
		regexMatched, err := regexp.MatchString(step.Line, matcher.currentLine)
		ProcessError(err, matcher.logger, "error matching line with regex")
		matched = regexMatched
	} else {
		matched = matcher.currentLine == step.Line
	}

	if matched {
		matcher.logger.Debugf("matched current line '%s' with step '%s'", matcher.currentLine, step.Line)
		if len(step.Input) > 0 {
			fmt.Print(step.Input + "\n")
			matcher.logger.Debugf("writing input '%s' to stdin", step.Input)
			_, err := matcher.stdin.Write([]byte(step.Input + "\n"))
			ProcessError(err, matcher.logger, "error writing to stdin")
		}
		return true
	}
	return false
}
