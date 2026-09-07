package play

import (
	"testing"

	cfg "github.com/rjnienaber/prescript/internal/config"
	"github.com/rjnienaber/prescript/internal/script"
	"github.com/rjnienaber/prescript/internal/utils"
	"github.com/stretchr/testify/assert"
)

// A run whose executable comes from the script, so each run in a multi-run
// script can point somewhere different.
func fixtureRun(t *testing.T, name string, fileName string, steps []script.Step) script.Run {
	return script.Run{
		Name:       name,
		Executable: getFixturePath(fileName, t),
		Steps:      steps,
	}
}

func multiRunConfig(t *testing.T) cfg.PlayConfig {
	config := createConfig(t, "fixtures/output.sh")
	config.Play.ExecutablePath = ""
	return config.Play
}

func TestRunsEveryRunInTheScript(t *testing.T) {
	t.Parallel()

	runs := []script.Run{
		fixtureRun(t, "first", "fixtures/output.sh", []script.Step{{Line: "Hello, world!"}}),
		fixtureRun(t, "second", "fixtures/input.sh", []script.Step{
			{Line: "Please enter your name: ", Input: "Richard"},
		}),
	}

	assert.Equal(t, utils.SUCCESS, RunAll(multiRunConfig(t), runs, &utils.CustomLogger{}))
}

// The point of the loop: a failing run must not stop the ones after it, or a
// comparison of three implementations stops telling you about two of them.
func TestALaterRunStillPlaysAfterAnEarlierOneFails(t *testing.T) {
	t.Parallel()

	played := fixtureRun(t, "third", "fixtures/output.sh", []script.Step{{Line: "Hello, world!"}})
	runs := []script.Run{
		fixtureRun(t, "first", "fixtures/output.sh", []script.Step{{Line: "not what is printed"}}),
		played,
	}

	result := RunAll(multiRunConfig(t), runs, &utils.CustomLogger{})

	assert.NotEqual(t, utils.SUCCESS, result)
	// The second run having succeeded on its own is what proves it was played:
	// it would have failed identically to the first had it been skipped.
	assert.Equal(t, utils.SUCCESS, RunAll(multiRunConfig(t), []script.Run{played}, &utils.CustomLogger{}))
}

func TestSingleRunStillReturnsItsOwnExitCode(t *testing.T) {
	t.Parallel()

	runs := []script.Run{fixtureRun(t, "only", "fixtures/output.sh", []script.Step{{Line: "nope"}})}

	assert.Equal(t, utils.CLI_ERROR, RunAll(multiRunConfig(t), runs, &utils.CustomLogger{}))
}

// A runner's flags belong to the interpreter and come first; what follows names
// the program, and the command line may still replace that half on its own.
func TestRunnerArgumentsComeFirst(t *testing.T) {
	t.Parallel()

	run := script.Run{RunnerArguments: []string{"-W0"}, Arguments: []string{"dice.rb"}}

	assert.Equal(t, []string{"-W0", "dice.rb"}, getArguments(cfg.PlayConfig{}, run))
	assert.Equal(t, []string{"-W0", "other.rb"},
		getArguments(cfg.PlayConfig{Arguments: []string{"other.rb"}}, run))
}

func TestArgumentsAreUnchangedWithoutARunner(t *testing.T) {
	t.Parallel()

	run := script.Run{Arguments: []string{"dice.bas"}}

	assert.Equal(t, []string{"dice.bas"}, getArguments(cfg.PlayConfig{}, run))
	assert.Equal(t, []string{"other.bas"},
		getArguments(cfg.PlayConfig{Arguments: []string{"other.bas"}}, run))
}
