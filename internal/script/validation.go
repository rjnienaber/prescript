package script

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	schema "github.com/xeipuuv/gojsonschema"
)

// validateTimeouts parses each step's timeout override. A duration that does
// not parse is reported here rather than at play time, alongside the regexes
// and for the same reason: a script that cannot be run is worth knowing about
// before a corpus run has spent twenty minutes reaching it.
func validateTimeouts(runs []Run) []string {
	var timeoutErrors []string
	for runIndex, run := range runs {
		for stepIndex := range run.Steps {
			step := &run.Steps[stepIndex]
			if step.Timeout == "" {
				continue
			}

			duration, err := time.ParseDuration(step.Timeout)
			switch {
			case err != nil:
				timeoutErrors = append(timeoutErrors,
					fmt.Sprintf("runs.%d.steps.%d.timeout: %s", runIndex, stepIndex, err.Error()))
			case duration <= 0:
				timeoutErrors = append(timeoutErrors,
					fmt.Sprintf("runs.%d.steps.%d.timeout: %q is not a positive duration", runIndex, stepIndex, step.Timeout))
			default:
				step.TimeoutDuration = duration
			}
		}
	}
	return timeoutErrors
}

func buildValidationErrors(resultErrors []schema.ResultError, regexErrors []string) error {
	validationErrors := []string{}
	for _, validationError := range resultErrors {
		validationErrors = append(validationErrors, validationError.String())
	}

	validationErrors = append(validationErrors, regexErrors...)

	// validation errors can be returned in random order so we order them
	sort.Strings(validationErrors)

	return validationError(validationErrors...)
}

// validationError gives every rejection of a script file the same shape, so a
// version problem reads like a schema problem rather than like a crash.
func validationError(messages ...string) error {
	return errors.New("Script validation errors:\n" + strings.Join(messages, "\n"))
}
