package lib

import (
	_ "embed"
	json2 "encoding/json"
	"errors"
	schema "github.com/xeipuuv/gojsonschema"
	"strings"
	"time"
)

//go:embed "script_schema.json"
var SCRIPT_SCHEMA []byte

type Script struct {
	Version string `json:"version"`
	Runs    []struct {
		Timestamp time.Time `json:"timestamp"`
		Command   string    `json:"command"`
		Args      []string  `json:"args"`
		ExitCode  int       `json:"exitCode"`
		Steps     []struct {
			Line    string `json:"line"`
			Input   string `json:"input,omitempty"`
			IsRegex bool   `json:"isRegex,omitempty"`
		} `json:"steps"`
	} `json:"runs"`
}

func NewScript(json string) (Script, error) {
	schemaLoader := schema.NewBytesLoader(SCRIPT_SCHEMA)
	jsonBytes := []byte(json)
	documentLoader := schema.NewBytesLoader(jsonBytes)
	result, err := schema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return Script{}, err
	}

	if result.Valid() {
		var script Script
		json2.Unmarshal([]byte(json), &script)
		return script, nil
	}

	validationErrors := []string{}
	for _, validationError := range result.Errors() {
		validationErrors = append(validationErrors, validationError.String())
	}

	err = errors.New("Script validation errors:\n" + strings.Join(validationErrors, "\n"))
	return Script{}, err
}
