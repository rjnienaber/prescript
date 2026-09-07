package script

import (
	_ "embed"
	json2 "encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
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

	directory, err := filepath.Abs(filepath.Dir(filePath))
	if err != nil {
		return Runner{}, err
	}

	runner, err = expandPlaceholders(runner, directory)
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

// runnerDir is the one placeholder a runner may use, and it stands for the
// directory the runner file itself was read from.
//
// A runner that activates a shim has to name a file — an interpreter flag like
// RUBYOPT=-r<file> is a path and nothing else. Written relative to the working
// directory that path holds only while prescript is run from one place, which
// is exactly what a corpus run does not do. Written absolute it holds only on
// the machine that wrote it. Resolved against the runner file, a runner and
// the shim it activates travel together and can be checked in.
const runnerDir = "runnerDir"

var placeholderPattern = regexp.MustCompile(`\$\{([^}]*)\}`)

// expandPlaceholders resolves ${runnerDir} everywhere a runner can name a
// path, and rejects anything else spelled the same way.
//
// Rejecting is the point: env values are handed to the child verbatim, with no
// shell anywhere to expand them, so a misspelled ${runnerDr} would otherwise
// reach the interpreter as a literal and be reported as a missing file
// somewhere far from the runner that named it.
func expandPlaceholders(runner Runner, directory string) (Runner, error) {
	var unknown []string

	expand := func(field, value string) string {
		return placeholderPattern.ReplaceAllStringFunc(value, func(match string) string {
			if match[2:len(match)-1] == runnerDir {
				return directory
			}

			unknown = append(unknown, fmt.Sprintf("%s: unknown placeholder %s, the only one is ${%s}", field, match, runnerDir))
			return match
		})
	}

	runner.Executable = expand("executable", runner.Executable)
	for i, argument := range runner.Arguments {
		runner.Arguments[i] = expand(fmt.Sprintf("arguments.%d", i), argument)
	}

	// Sorted, because a map's order is not one: two runs of the same broken
	// runner should be able to report the same thing.
	for _, name := range sortedNames(runner.Env) {
		runner.Env[name] = expand("env."+name, runner.Env[name])
	}

	if len(unknown) > 0 {
		return Runner{}, validationError(unknown...)
	}

	return runner, nil
}

func sortedNames(env map[string]string) []string {
	names := make([]string, 0, len(env))
	for name := range env {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
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
