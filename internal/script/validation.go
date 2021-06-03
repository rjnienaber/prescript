package script

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	schema "github.com/xeipuuv/gojsonschema"
)

func validateRegexes(runs []Run) []string {
	var regexErrors []string
	// validate regexes should they exist
	for runIndex, run := range runs {
		for stepIndex := range run.Steps {
			step := &run.Steps[stepIndex]
			if step.IsRegex {
				regex, err := regexp.Compile(step.Line)
				if err != nil {
					regexError := fmt.Sprintf("runs.%d.steps.%d.line: %s", runIndex, stepIndex, err.Error())
					regexErrors = append(regexErrors, regexError)
				} else {
					step.LineRegex = *regex
				}
			}
		}
	}
	return regexErrors
}

func buildValidationErrors(resultErrors []schema.ResultError, regexErrors []string) error {
	validationErrors := []string{}
	for _, validationError := range resultErrors {
		validationErrors = append(validationErrors, validationError.String())
	}

	validationErrors = append(validationErrors, regexErrors...)

	// validation errors can be returned in random order so we order them
	sort.Strings(validationErrors)

	err := errors.New("Script validation errors:\n" + strings.Join(validationErrors, "\n"))
	return err
}
