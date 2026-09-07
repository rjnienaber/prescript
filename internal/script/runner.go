package script

import (
	_ "embed"
	json2 "encoding/json"
	"path/filepath"
	"strings"

	schema "github.com/xeipuuv/gojsonschema"
)

//go:embed "runner_schema.json"
var RunnerSchemaBytes []byte

// Runner is how to launch an implementation, as opposed to which program to
// launch and what it should print. A runner is written once per language and a
// script once per program, which is what keeps a corpus of a hundred programs
// in a dozen languages at roughly a hundred files rather than twelve hundred.
type Runner struct {
	Version    string            `json:"version"`
	Name       string            `json:"name,omitempty"`
	Executable string            `json:"executable"`
	Arguments  []string          `json:"arguments,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
	InheritEnv bool              `json:"inheritEnv,omitempty"`
	Terminal   string            `json:"terminal,omitempty"`
}

func ParseRunnerFromFile(filePath string) (Runner, error) {
	contents, err := readScriptFile(filePath)
	if err != nil {
		return Runner{}, err
	}

	runner, err := ParseRunnerFromBytes(contents)
	if err != nil {
		return Runner{}, err
	}

	if runner.Name == "" {
		name := filepath.Base(filePath)
		runner.Name = strings.TrimSuffix(name, filepath.Ext(name))
	}

	return runner, nil
}

func ParseRunnerFromBytes(json []byte) (Runner, error) {
	// Same order as a script, and for the same reason: a runner from a newer
	// prescript should be told to upgrade, not handed a list of fields it does
	// not recognise.
	if err := checkDeclaredVersion(json); err != nil {
		return Runner{}, err
	}

	result, err := schema.Validate(schema.NewBytesLoader(RunnerSchemaBytes), schema.NewBytesLoader(json))
	if err != nil {
		return Runner{}, err
	}

	if !result.Valid() {
		return Runner{}, buildValidationErrors(result.Errors(), nil)
	}

	var runner Runner
	if err := json2.Unmarshal(json, &runner); err != nil {
		return Runner{}, err
	}

	return runner, nil
}

// ApplyRunner merges a runner's launch details into every run of a script. The
// two describe different halves of one command line: the runner says how to
// start a Ruby program, the script says which program and what it should print.
func ApplyRunner(runs []Run, runner Runner) []Run {
	merged := make([]Run, len(runs))

	for i, run := range runs {
		if runner.Executable != "" {
			run.Executable = runner.Executable
		}

		run.RunnerArguments = runner.Arguments
		run.Env = mergeEnv(runner.Env, run.Env)
		run.InheritEnv = run.InheritEnv || runner.InheritEnv

		// As with env, the script wins: it describes this program, while the
		// runner describes every program written in one language.
		if run.Terminal == "" {
			run.Terminal = runner.Terminal
		}

		merged[i] = run
	}

	return merged
}

// mergeEnv layers a script's env over a runner's. The two are usually about
// different things — a language's RUBYOPT against a program's own seed — but
// where they collide the script wins: it is the more specific description of
// the case being run.
//
// Both being absent stays absent, because that is what means "inherit".
func mergeEnv(runner map[string]string, run map[string]string) map[string]string {
	if runner == nil && run == nil {
		return nil
	}

	merged := map[string]string{}
	for name, value := range runner {
		merged[name] = value
	}
	for name, value := range run {
		merged[name] = value
	}

	return merged
}
