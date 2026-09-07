package play

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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
// Node's is the generator in runners/node/random.js.
var seededRunners = []struct {
	runner  string
	program string
	drawn   string
}{
	{runner: "ruby.yaml", program: "rng.rb", drawn: "548 715 602"},
	{runner: "python.yaml", program: "rng.py", drawn: "844 757 420"},
	{runner: "node.yaml", program: "rng.js", drawn: "358 105 675"},
	{runner: "go.yaml", program: "rng.go", drawn: "604 940 664"},
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
//
// Somewhere has to actually run them, though, or a shipped runner is only ever
// claimed to work. Setting PRESCRIPT_REQUIRE_RUNNERS turns a missing language
// into a failure, and CI sets it: an image that stops installing Ruby should
// turn the build red rather than quietly shrink what is being checked.
func loadRunner(t *testing.T, name string) script.Runner {
	t.Helper()
	runner, err := script.ParseRunnerFromFile(repositoryPath(t, "runners", name))
	assert.NoError(t, err)

	if _, err := exec.LookPath(runner.Executable); err != nil {
		if os.Getenv("PRESCRIPT_REQUIRE_RUNNERS") != "" {
			t.Fatalf("%s is not installed, so %s cannot be checked, and PRESCRIPT_REQUIRE_RUNNERS says it has to be", runner.Executable, name)
		}
		t.Skipf("%s is not installed, so %s cannot be checked here", runner.Executable, name)
	}

	return runner
}

// drawingRun scripts one of the rng fixtures: it is asked for as many numbers
// as `drawn` names, and has to print exactly those.
func drawingRun(t *testing.T, program string, drawn string) script.Run {
	t.Helper()
	return script.Run{
		Arguments: []string{repositoryPath(t, "test", "fixtures", "rng", program)},
		ExitCode:  0,
		Steps: []script.Step{
			{Line: "HOW MANY? ", Input: strconv.Itoa(len(strings.Fields(drawn)))},
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
			assert.Equal(t, 0, Run(drawingConfig(), runs[0], logger).ExitCode)

			// Twice, because "deterministic" is a claim about runs and not
			// about one run.
			assert.Equal(t, 0, Run(drawingConfig(), runs[0], logger).ExitCode)
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
				exitCode = Run(drawingConfig(), run, &utils.CustomLogger{}).ExitCode
			})

			assert.NotEqual(t, 0, exitCode, "expected unseeded output to differ from the recorded draw")
			assert.Contains(t, report, "exited-early: step 2 of 2 did not match")
		})
	}
}

// tapedRunners are the languages whose uniform generator a tape can be fed
// into. Go is missing on purpose: math/rand's package-level functions draw
// from a source that cannot be replaced from outside the program, so feeding
// Go a tape needs the build to cooperate rather than the launch. See
// docs/determinism.md.
var tapedRunners = []struct {
	runner  string
	program string
}{
	{runner: "ruby.yaml", program: "rng.rb"},
	{runner: "python.yaml", program: "rng.py"},
	{runner: "node.yaml", program: "rng.js"},
}

// referenceDraws is what the reference BASIC interpreter itself prints for
// INT(RND(1) * 1000), five times, at seed 0. Every port replaying the tape
// should print the same, because it is drawing the same numbers.
const referenceDraws = "973 411 47 33 295"

func tapePath(t *testing.T) string {
	t.Helper()
	return repositoryPath(t, "tapes", "vintbas-seed0.tape")
}

func tapedConfig(t *testing.T) cfg.PlayConfig {
	t.Helper()
	config := drawingConfig()
	config.Tape = tapePath(t)
	return config
}

// The whole point of a tape: one script, one expected transcript, every port.
// Seeding cannot do this -- each language draws a different sequence from the
// same seed -- so without a tape this test would need three expectations.
func TestOneTapeGivesEveryLanguageTheSameNumbers(t *testing.T) {
	for _, language := range tapedRunners {
		t.Run(language.runner, func(t *testing.T) {
			t.Parallel()
			runner := loadRunner(t, language.runner)
			runs := script.ApplyRunner([]script.Run{drawingRun(t, language.program, referenceDraws)}, runner)

			assert.Equal(t, 0, Run(tapedConfig(t), runs[0], &utils.CustomLogger{}).ExitCode)
		})
	}
}

// And that those numbers are the reference's, not merely numbers the ports
// happen to agree on. Reading it from the tape rather than hard-coding it
// means a regenerated tape cannot silently disagree with the expectation
// above.
func TestTheTapeHoldsTheReferencesOwnDraws(t *testing.T) {
	t.Parallel()
	contents, err := os.ReadFile(tapePath(t))
	assert.NoError(t, err)

	var drawn []string
	for _, line := range strings.Split(string(contents), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		value, err := strconv.ParseFloat(line, 64)
		assert.NoError(t, err)

		// INT(RND(1) * 1000), which is what the fixtures compute.
		drawn = append(drawn, strconv.Itoa(int(value*1000)))
		if len(drawn) == 5 {
			break
		}
	}

	assert.Equal(t, referenceDraws, strings.Join(drawn, " "))
}

// A port that draws more randomness than the reference did has restructured
// how it consumes it, which is a finding rather than something to paper over
// by starting the tape again.
func TestRunningOffTheEndOfTheTapeFails(t *testing.T) {
	short := filepath.Join(t.TempDir(), "short.tape")
	assert.NoError(t, os.WriteFile(short, []byte("# two values, and the program wants five\n.9732723\n.41177148\n"), 0o600))

	for _, language := range tapedRunners {
		t.Run(language.runner, func(t *testing.T) {
			runner := loadRunner(t, language.runner)
			runs := script.ApplyRunner([]script.Run{drawingRun(t, language.program, referenceDraws)}, runner)

			config := drawingConfig()
			config.Tape = short

			exitCode := 0
			report := captureStderr(t, func() {
				exitCode = Run(config, runs[0], &utils.CustomLogger{}).ExitCode
			})

			assert.NotEqual(t, 0, exitCode)
			assert.Contains(t, report, "the tape ran out after 2 values")
		})
	}
}

// How much of the tape a port used is what separates a real logic bug from a
// port that restructured its draws, so a shim writes it down when asked.
func TestAPortRecordsHowMuchOfTheTapeItUsed(t *testing.T) {
	for _, language := range tapedRunners {
		t.Run(language.runner, func(t *testing.T) {
			t.Parallel()
			runner := loadRunner(t, language.runner)
			runs := script.ApplyRunner([]script.Run{drawingRun(t, language.program, referenceDraws)}, runner)

			usage := filepath.Join(t.TempDir(), "usage")
			runs[0].Env["PRESCRIPT_TAPE_USAGE"] = usage

			assert.Equal(t, 0, Run(tapedConfig(t), runs[0], &utils.CustomLogger{}).ExitCode)

			drawn, err := os.ReadFile(usage)
			assert.NoError(t, err)
			assert.Equal(t, "5\n", string(drawn))
		})
	}
}

// A tape that is not there is a mistake in the command line, and a program
// that finds out halfway through has already printed half a transcript that
// means nothing.
func TestAMissingTapeIsReportedBeforeTheRun(t *testing.T) {
	config := drawingConfig()
	config.Tape = filepath.Join(t.TempDir(), "absent.tape")

	run := drawingRun(t, "rng.rb", referenceDraws)
	run.Executable = "ruby"

	exitCode := 0
	report := captureStderr(t, func() {
		exitCode = Run(config, run, &utils.CustomLogger{}).ExitCode
	})

	assert.Equal(t, utils.USER_ERROR, exitCode)
	assert.Contains(t, report, "could not read the tape")
}
