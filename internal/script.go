package internal

import (
	_ "embed"
	json2 "encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"regexp"
	"sort"
	"strings"
	"time"

	schema "github.com/xeipuuv/gojsonschema"
)

//go:embed "script_schema.json"
var ScriptSchema []byte

type Step struct {
	Line      string `json:"line"`
	LineRegex regexp.Regexp
	Input     string `json:"input,omitempty"`
	IsRegex   bool   `json:"isRegex,omitempty"`
}

type Run struct {
	Timestamp  time.Time `json:"timestamp"`
	Executable string    `json:"executable"`
	Arguments  []string  `json:"arguments"`
	ExitCode   int       `json:"exitCode"`
	Steps      []Step    `json:"steps"`
}

type Script struct {
	Version string `json:"version"`
	Runs    []Run  `json:"runs"`
}

func validateRegexes(runs []Run) []string {
	var regexErrors []string
	// validate regexes should they exist
	for runIndex, run := range runs {
		for stepIndex, step := range run.Steps {
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

func NewScriptFromFile(filePath string) (Script, error) {
	json, err := ioutil.ReadFile(filePath)
	if err != nil {
		return Script{}, err
	}
	return NewScriptFromBytes(json)
}

func NewScriptFromBytes(json []byte) (Script, error) {
	schemaLoader := schema.NewBytesLoader(ScriptSchema)
	documentLoader := schema.NewBytesLoader(json)
	result, err := schema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return Script{}, err
	}

	var regexErrors []string
	if result.Valid() {
		var script Script
		err = json2.Unmarshal(json, &script)

		if err != nil {
			return Script{}, err
		}

		regexErrors = validateRegexes(script.Runs)
		if len(regexErrors) == 0 {
			return script, err
		}
	}

	validationErrors := []string{}
	for _, validationError := range result.Errors() {
		validationErrors = append(validationErrors, validationError.String())
	}

	validationErrors = append(validationErrors, regexErrors...)

	// validation errors can be returned in random order so we order them
	sort.Strings(validationErrors)

	err = errors.New("Script validation errors:\n" + strings.Join(validationErrors, "\n"))
	return Script{}, err
}
