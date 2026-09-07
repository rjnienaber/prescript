package script

import (
	_ "embed"
	json2 "encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rjnienaber/prescript/internal/config"
	"github.com/rjnienaber/prescript/internal/utils"
	schema "github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

//go:embed "script_schema.json"
var SchemaBytes []byte

// ParseScriptFromFile reads a script written as either JSON or YAML, chosen by
// the file's extension. YAML is converted to JSON and validated by the same
// schema against the same Go types, so the two cannot drift apart: there is one
// description of the format and one set of error messages, whichever a script
// happens to be written in.
func ParseScriptFromFile(filePath string) (Script, error) {
	contents, err := readScriptFile(filePath)
	if err != nil {
		return Script{}, err
	}

	return ParseScriptFromBytes(contents)
}

// readScriptFile reads a script or a runner, as YAML when the extension says
// so and as JSON otherwise, and hands back JSON either way.
func readScriptFile(filePath string) ([]byte, error) {
	contents, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".yaml", ".yml":
		contents, err = yamlToJson(contents)
		if err != nil {
			return nil, fmt.Errorf("could not read %s as YAML: %w", filePath, err)
		}
	}

	return contents, nil
}

func yamlToJson(document []byte) ([]byte, error) {
	var content any
	if err := yaml.Unmarshal(document, &content); err != nil {
		return nil, err
	}

	converted, err := json2.Marshal(content)
	if err != nil {
		// The case worth naming: YAML allows a mapping key of any type, JSON
		// does not, and the error from encoding/json says only that it met a
		// map[interface{}]interface{}.
		return nil, fmt.Errorf("%w; every key in a script has to be a string", err)
	}

	return converted, nil
}

func ParseScriptFromBytes(json []byte) (Script, error) {
	// Checked before the schema deliberately. A script from a newer prescript
	// otherwise fails as a list of unrecognised fields, which states the same
	// problem far less usefully than naming the version does.
	if err := checkDeclaredVersion(json); err != nil {
		return Script{}, err
	}

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

		regexErrors = compileRedactions(script)
		regexErrors = append(regexErrors, validateTimeouts(script.Runs)...)
		regexErrors = append(regexErrors, ensureNames(script.Runs)...)
		if len(regexErrors) == 0 {
			return script, err
		}
	}

	err = buildValidationErrors(result.Errors(), regexErrors)
	return Script{}, err
}

// checkDeclaredVersion reads just enough of the document to find its version.
// Anything that will not unmarshal, or that omits the version entirely, is left
// to the schema, which describes shape problems better than this can.
func checkDeclaredVersion(document []byte) error {
	var declared struct {
		Version string `json:"version"`
	}

	if err := json2.Unmarshal(document, &declared); err != nil || declared.Version == "" {
		return nil
	}

	if err := checkVersion(declared.Version); err != nil {
		return validationError(err.Error())
	}
	return nil
}

func BuildScriptJson(cfg config.RecordConfig, lines []utils.CapturedLine, exitCode int, now time.Time, logger utils.Logger) (string, error) {
	var steps []Step
	if cfg.DontCompress {
		steps = dontCompressCapturedLines(lines)
	} else {
		steps = compressCaputuredLines(lines)
	}

	script := Script{
		Version: CurrentVersion,
		Runs: []Run{
			{
				Timestamp:  now,
				Executable: cfg.ExecutablePath,
				Arguments:  cfg.Arguments,
				ExitCode:   exitCode,
				Steps:      steps,

				// Only when it is not the default. A recording made through
				// pipes has to say so, or replaying it would give the program
				// a terminal it was never recorded against.
				Terminal: pipesOnly(cfg.Terminal),
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

// pipesOnly keeps "pipes" and drops everything else, so an omitted or
// explicit "pty" both come out as the default rather than as noise in the file.
func pipesOnly(terminal string) string {
	if terminal == string(utils.TerminalPipes) {
		return terminal
	}
	return ""
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

// ensureNames gives every run a name and rejects a script in which two runs
// end up with the same one. A name is how a run is identified in the output of
// a multi-run script, so two runs called the same thing makes that output
// ambiguous precisely when there is something to tell apart. Generated names
// are checked alongside declared ones: a run explicitly called "run 1" collides
// with the unnamed run at index 1.
func ensureNames(runs []Run) []string {
	var nameErrors []string
	seen := map[string]int{}

	for i := range runs {
		run := &runs[i]
		if run.Name == "" {
			run.Name = "run " + strconv.Itoa(i)
		}

		if first, taken := seen[run.Name]; taken {
			nameErrors = append(nameErrors, fmt.Sprintf("runs.%d.name: duplicate run name %s, already used by runs.%d", i, strconv.Quote(run.Name), first))
			continue
		}
		seen[run.Name] = i
	}

	return nameErrors
}
