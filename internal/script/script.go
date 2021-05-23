package script

import (
	_ "embed"
	json2 "encoding/json"
	"io/ioutil"
	"time"

	"github.com/rjnienaber/prescript/internal/config"
	"github.com/rjnienaber/prescript/internal/utils"
	schema "github.com/xeipuuv/gojsonschema"
)

//go:embed "script_schema.json"
var SchemaBytes []byte

func NewScriptFromFile(filePath string) (Script, error) {
	json, err := ioutil.ReadFile(filePath)
	if err != nil {
		return Script{}, err
	}
	return NewScriptFromBytes(json)
}

func NewScriptFromBytes(json []byte) (Script, error) {
	schemaLoader := schema.NewBytesLoader(SchemaBytes)
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

	err = buildValidationErrors(result.Errors(), regexErrors)
	return Script{}, err
}

func BuildScriptJson(cfg config.RecordConfig, lines []utils.CapturedLine, exitCode int, now time.Time, logger utils.Logger) (string, error) {
	steps := []Step{}
	for i, line := range lines {
		if line.LineType == utils.Input {
			output := lines[i-1].Value
			steps = append(steps, Step{Line: output, Input: line.Value, IsRegex: false})
		}
	}

	script := Script{
		Version: "0.1",
		Runs: []Run{
			{
				Timestamp:  now,
				Executable: cfg.ExecutablePath,
				Arguments:  cfg.Arguments,
				ExitCode:   exitCode,
				Steps:      steps,
			},
		},
	}

	scriptBytes, err := json2.MarshalIndent(script, "", "  ")
	if err != nil {
		logger.Error("unable to convert script file to string", err)
		return "", err
	}

	return string(scriptBytes), nil
}
