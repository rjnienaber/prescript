package lib

import (
	_ "embed"
	json2 "encoding/json"
	"errors"
	schema "github.com/xeipuuv/gojsonschema"
	"sort"
	"strings"
	"time"
)

//go:embed "script_schema.json"
var SCRIPT_SCHEMA []byte

type Script struct {
	Version string `json:"version"`
	Runs    []struct {
		Timestamp  time.Time `json:"timestamp"`
		Executable string    `json:"executable"`
		Arguments  []string  `json:"arguments"`
		ExitCode   int       `json:"exitCode"`
		Steps      []struct {
			Line    string `json:"line"`
			Input   string `json:"input,omitempty"`
			IsRegex bool   `json:"isRegex,omitempty"`
		} `json:"steps"`
	} `json:"runs"`
}

func NewScript(json string) (Script, error) {
	schemaLoader := schema.NewBytesLoader(SCRIPT_SCHEMA)
	documentLoader := schema.NewStringLoader(json)
	result, err := schema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return Script{}, err
	}

	if result.Valid() {
		var script Script
		err = json2.Unmarshal([]byte(json), &script)
		return script, err
	}

	validationErrors := []string{}
	for _, validationError := range result.Errors() {
		validationErrors = append(validationErrors, validationError.String())
	}

	// validation errors can be returned in random order so we order them
	sort.Strings(validationErrors)

	err = errors.New("Script validation errors:\n" + strings.Join(validationErrors, "\n"))
	return Script{}, err
}
