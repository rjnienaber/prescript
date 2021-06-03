package play

import (
	"fmt"
	"github.com/rjnienaber/prescript/internal/script"
	"github.com/rjnienaber/prescript/internal/utils"
	"io"
)

type StepMatcher struct {
	currentLine      string
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

func (matcher *StepMatcher) NextExpectedLine() string {
	return matcher.steps[matcher.currentStepIndex].Line
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
			matcher.ResetLine()
		}
	}
	return nil
}

func (matcher *StepMatcher) matchLine(step script.Step) (bool, error) {
	var matched bool
	if step.IsRegex {
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
