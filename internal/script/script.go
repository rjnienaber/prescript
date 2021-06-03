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

func ParseScriptFromFile(filePath string) (Script, error) {
	json, err := ioutil.ReadFile(filePath)
	if err != nil {
		return Script{}, err
	}
	return ParseScriptFromBytes(json)
}

func ParseScriptFromBytes(json []byte) (Script, error) {
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
	var steps []Step
	if cfg.DontCompress {
		steps = dontCompressCapturedLines(lines)
	} else {
		steps = compressCaputuredLines(lines)
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

func compressCaputuredLines(lines []utils.CapturedLine) []Step {
	steps := []Step{}
	for i, line := range lines {
		if line.LineType == utils.Input {
			output := lines[i-1].Value
			steps = append(steps, Step{Line: output, Input: line.Value})
		}
	}
	return steps
}

func dontCompressCapturedLines(lines []utils.CapturedLine) []Step {
	steps := []Step{}
	for i, line := range lines {
		if line.LineType == utils.Input {
			// if first line and it's input, special case
			if i == 0 {
				steps = append(steps, Step{Input: line.Value})
			}
			continue
		}

		// process output line
		if (i + 1) < len(lines) {
			nextLine := lines[i+1]
			if nextLine.LineType == utils.Input {
				steps = append(steps, Step{Line: line.Value, Input: nextLine.Value})
				continue
			}
		}

		if line.Value == "" {
			continue
		}

		steps = append(steps, Step{Line: line.Value})
	}

	return steps
}
