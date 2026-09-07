package play

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	cfg "github.com/rjnienaber/prescript/internal/config"
	"github.com/rjnienaber/prescript/internal/script"
	"github.com/rjnienaber/prescript/internal/utils"
	"github.com/stretchr/testify/assert"
)

// The shipped runners, played against a program that does nothing but draw
// random numbers. Each language draws a different sequence from the same seed
// -- that is what a tape fixes and a seed does not -- so the numbers below
// belong to the language rather than to the program.
//
// They are the language's documented sequence for that seed, not a snapshot of
// one machine's: Ruby and Python both promise a reproducible Mersenne Twister,
// Go's is the fixed source it used before it started seeding itself, and
// Node's is the generator in runners/node/seed.js.
var seededRunners = []struct {
	runner  string
	program string
	drawn   string
}{
	{runner: "ruby.yaml", program: "rng.rb", drawn: "684 559 629"},
	{runner: "python.yaml", program: "rng.py", drawn: "864 394 776"},
	{runner: "node.yaml", program: "rng.js", drawn: "358 105 675"},
	{runner: "go.yaml", program: "rng.go", drawn: "81 887 847"},
}

func repositoryPath(t *testing.T, parts ...string) string {
	t.Helper()
	cwd, err := os.Getwd()
	assert.NoError(t, err)

	path, err := filepath.Abs(filepath.Join(append([]string{cwd, "..", ".."}, parts...)...))
	assert.NoError(t, err)
	return path
}

// loadRunner skips the test when the language is not installed. A machine
// without Node should report that these tests did not run, not that they
// failed: the runner is a claim about Node, and there is no Node to check it
// against.
func loadRunner(t *testing.T, name string) script.Runner {
	t.Helper()
	runner, err := script.ParseRunnerFromFile(repositoryPath(t, "runners", name))
	assert.NoError(t, err)

	if _, err := exec.LookPath(runner.Executable); err != nil {
		t.Skipf("%s is not installed, so %s cannot be checked here", runner.Executable, name)
	}

	return runner
}

func drawingRun(t *testing.T, program string, drawn string) script.Run {
	t.Helper()
	return script.Run{
		Arguments: []string{repositoryPath(t, "test", "fixtures", "rng", program)},
		ExitCode:  0,
		Steps: []script.Step{
			{Line: "HOW MANY? ", Input: "3"},
			{Line: drawn},
		},
	}
}

func drawingConfig() cfg.PlayConfig {
	// Generous, because `go run` compiles before it runs and a cold module
	// cache is slower than anything else these tests do.
	return cfg.PlayConfig{Timeout: getTimeout(60000), Quiet: true}
}

// The point of the whole exercise: a program whose output is random becomes a
// program whose output can be written down.
func TestRunnersMakeRandomnessRepeatable(t *testing.T) {
	for _, language := range seededRunners {
		t.Run(language.runner, func(t *testing.T) {
			t.Parallel()
			runner := loadRunner(t, language.runner)
			runs := script.ApplyRunner([]script.Run{drawingRun(t, language.program, language.drawn)}, runner)

			logger := &utils.CustomLogger{}
			assert.Equal(t, 0, Run(drawingConfig(), runs[0], logger))

			// Twice, because "deterministic" is a claim about runs and not
			// about one run.
			assert.Equal(t, 0, Run(drawingConfig(), runs[0], logger))
		})
	}
}

// And the other half of the claim: that it is the runner doing it. Without one
// the same script fails, which is what says the shim is load-bearing rather
// than decorative.
func TestWithoutARunnerTheSameProgramIsUnscriptable(t *testing.T) {
	// Not parallel: capturing the report means swapping os.Stderr, which is
	// one process-wide thing however many tests want it.
	for _, language := range seededRunners {
		t.Run(language.runner, func(t *testing.T) {
			runner := loadRunner(t, language.runner)

			run := drawingRun(t, language.program, language.drawn)
			run.Executable = runner.Executable
			run.RunnerArguments = runner.Arguments

			exitCode := 0
			report := captureStderr(t, func() {
				exitCode = Run(drawingConfig(), run, &utils.CustomLogger{})
			})

			assert.NotEqual(t, 0, exitCode, "expected unseeded output to differ from the recorded draw")
			assert.Contains(t, report, "exited-early: step 2 of 2 did not match")
		})
	}
}
