package script

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

const rubyRunner = `# One runner per language, shared by every script.
version: "0.5"
executable: ruby
arguments: ["-W0"]
env:
  RUBYOPT: "--disable-gems"
`

func TestParsesRunner(t *testing.T) {
	runner, err := ParseRunnerFromFile(writeScript(t, "ruby.yaml", rubyRunner))
	assert.NoError(t, err)
	assert.Equal(t, "ruby", runner.Executable)
	assert.Equal(t, []string{"-W0"}, runner.Arguments)
	assert.Equal(t, map[string]string{"RUBYOPT": "--disable-gems"}, runner.Env)
}

// A runner is identified in output by name, and a file already has one.
func TestRunnerIsNamedAfterItsFile(t *testing.T) {
	runner, err := ParseRunnerFromFile(writeScript(t, "haskell.yaml", "version: \"0.5\"\nexecutable: runghc\n"))
	assert.NoError(t, err)
	assert.Equal(t, "haskell", runner.Name)
}

func TestDeclaredRunnerNameWins(t *testing.T) {
	runner, err := ParseRunnerFromFile(writeScript(t, "r.yaml", "version: \"0.5\"\nname: ruby 3.3\nexecutable: ruby\n"))
	assert.NoError(t, err)
	assert.Equal(t, "ruby 3.3", runner.Name)
}

func TestRunnerNeedsAnExecutable(t *testing.T) {
	_, err := ParseRunnerFromBytes([]byte(`{"version": "0.5"}`))
	assert.Error(t, err)
	expected := `Script validation errors:
(root): executable is required`
	assert.Equal(t, expected, err.Error())
}

func TestRunnerRejectsUnknownFields(t *testing.T) {
	_, err := ParseRunnerFromBytes([]byte(`{"version": "0.5", "executable": "ruby", "steps": []}`))
	assert.Error(t, err)
	expected := `Script validation errors:
(root): Additional property steps is not allowed`
	assert.Equal(t, expected, err.Error())
}

// The version is checked the same way a script's is, so a runner written for a
// newer prescript says to upgrade rather than listing fields it cannot read.
func TestRunnerVersionIsChecked(t *testing.T) {
	_, err := ParseRunnerFromBytes([]byte(`{"version": "9.9", "executable": "ruby"}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "upgrade prescript")
}

func runnerFor(t *testing.T, runs []Run, runner Runner) Run {
	t.Helper()
	merged := ApplyRunner(runs, runner)
	assert.Len(t, merged, len(runs))
	return merged[0]
}

func TestApplyRunnerReplacesTheExecutable(t *testing.T) {
	run := runnerFor(t, []Run{{Executable: "vintbas"}}, Runner{Executable: "ruby"})
	assert.Equal(t, "ruby", run.Executable)
}

func TestApplyRunnerKeepsRunnerArgumentsSeparate(t *testing.T) {
	// The runner's flags belong to the interpreter and the run's arguments name
	// the program, so they must not be flattened into one list here: the
	// command line can still replace the second half on its own.
	run := runnerFor(t, []Run{{Arguments: []string{"dice.rb"}}}, Runner{Arguments: []string{"-W0"}})
	assert.Equal(t, []string{"-W0"}, run.RunnerArguments)
	assert.Equal(t, []string{"dice.rb"}, run.Arguments)
}

func TestApplyRunnerMergesEnv(t *testing.T) {
	run := runnerFor(t,
		[]Run{{Env: map[string]string{"SEED": "0"}}},
		Runner{Env: map[string]string{"RUBYOPT": "--disable-gems"}})
	assert.Equal(t, map[string]string{"SEED": "0", "RUBYOPT": "--disable-gems"}, run.Env)
}

// Where the two name the same variable the script wins: it describes the case
// being run, and the runner only describes how the language is started.
func TestScriptEnvWinsOverRunnerEnv(t *testing.T) {
	run := runnerFor(t,
		[]Run{{Env: map[string]string{"TZ": "UTC"}}},
		Runner{Env: map[string]string{"TZ": "Europe/London"}})
	assert.Equal(t, "UTC", run.Env["TZ"])
}

// Neither declaring an env has to stay absent, because absent is what means
// "inherit"; an empty map would mean an empty environment instead.
func TestNoEnvOnEitherSideStaysAbsent(t *testing.T) {
	run := runnerFor(t, []Run{{}}, Runner{Executable: "ruby"})
	assert.Nil(t, run.Env)
	assert.Nil(t, run.Environment([]string{"HOME=/home/richard"}))
}

func TestInheritEnvFromEitherSide(t *testing.T) {
	fromRunner := runnerFor(t, []Run{{Env: map[string]string{"A": "1"}}}, Runner{InheritEnv: true})
	assert.True(t, fromRunner.InheritEnv)

	fromRun := runnerFor(t, []Run{{Env: map[string]string{"A": "1"}, InheritEnv: true}}, Runner{})
	assert.True(t, fromRun.InheritEnv)
}

func TestApplyRunnerLeavesTheOriginalRunsAlone(t *testing.T) {
	runs := []Run{{Executable: "vintbas", Env: map[string]string{"SEED": "0"}}}
	ApplyRunner(runs, Runner{Executable: "ruby", Env: map[string]string{"RUBYOPT": "-W0"}})

	assert.Equal(t, "vintbas", runs[0].Executable)
	assert.Equal(t, map[string]string{"SEED": "0"}, runs[0].Env)
}

func TestApplyRunnerSuppliesTheTerminal(t *testing.T) {
	run := runnerFor(t, []Run{{}}, Runner{Terminal: "pipes"})
	assert.Equal(t, "pipes", run.Terminal)
}

// As with env, the script is the more specific description: it is about this
// one program, while the runner is about every program written in a language.
func TestScriptTerminalWinsOverRunnerTerminal(t *testing.T) {
	run := runnerFor(t, []Run{{Terminal: "pty"}}, Runner{Terminal: "pipes"})
	assert.Equal(t, "pty", run.Terminal)
}

func TestRunnerDirResolvesToTheRunnerFilesDirectory(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	path := filepath.Join(directory, "ruby.yaml")
	writeRunner(t, path, `version: "0.8"
executable: ruby
arguments: ["-I${runnerDir}/lib"]
env:
  RUBYOPT: "-r${runnerDir}/ruby/seed.rb"
`)

	runner, err := ParseRunnerFromFile(path)
	assert.NoError(t, err)

	// Absolute, because the point of the placeholder is a path that survives
	// prescript being run from somewhere else.
	resolved, err := filepath.Abs(directory)
	assert.NoError(t, err)
	assert.Equal(t, "-r"+resolved+"/ruby/seed.rb", runner.Env["RUBYOPT"])
	assert.Equal(t, []string{"-I" + resolved + "/lib"}, runner.Arguments)
}

func TestRunnerDirResolvesInTheExecutable(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	path := filepath.Join(directory, "wrapped.yaml")
	writeRunner(t, path, `version: "0.8"
executable: "${runnerDir}/run.sh"
`)

	runner, err := ParseRunnerFromFile(path)
	assert.NoError(t, err)

	resolved, err := filepath.Abs(directory)
	assert.NoError(t, err)
	assert.Equal(t, resolved+"/run.sh", runner.Executable)
}

// A misspelled placeholder would otherwise reach the interpreter as a literal
// and be reported as a missing file, a long way from the runner that named it.
func TestUnknownPlaceholderIsRejected(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	path := filepath.Join(directory, "ruby.yaml")
	writeRunner(t, path, `version: "0.8"
executable: ruby
env:
  RUBYOPT: "-r${runnerDr}/seed.rb"
`)

	_, err := ParseRunnerFromFile(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "env.RUBYOPT: unknown placeholder ${runnerDr}")
	assert.Contains(t, err.Error(), "the only one is ${runnerDir}")
}

func TestARunnerWithoutPlaceholdersIsUntouched(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	path := filepath.Join(directory, "ruby.yaml")
	writeRunner(t, path, `version: "0.8"
executable: ruby
env:
  PRESCRIPT_SEED: "0"
`)

	runner, err := ParseRunnerFromFile(path)
	assert.NoError(t, err)
	assert.Equal(t, "ruby", runner.Executable)
	assert.Equal(t, "0", runner.Env["PRESCRIPT_SEED"])
}

func writeRunner(t *testing.T, path string, contents string) {
	t.Helper()
	assert.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
}
