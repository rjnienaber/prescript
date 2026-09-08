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
// Go's is the fixed source it used before it started seeding itself, Java's is
// the linear congruential generator java.util.Random specifies exactly, and
// Node's is the generator in runners/node/random.js.
//
// .NET draws what Node draws, and that is not a mistake. Its shim replaces
// System.Random outright, so it has to bring a generator rather than seed one,
// and it brings the same mulberry32 the Node shim already carries instead of a
// second one to explain. Two languages agreeing under a seed is a coincidence
// of the shims; a tape is what makes agreement mean something.
var seededRunners = []struct {
	runner  string
	program string
	drawn   string
}{
	{runner: "ruby.yaml", program: "rng.rb", drawn: "548 715 602"},
	{runner: "python.yaml", program: "rng.py", drawn: "844 757 420"},
	{runner: "node.yaml", program: "rng.js", drawn: "358 105 675"},
	{runner: "go.yaml", program: "rng.go", drawn: "604 940 664"},
	{runner: "java.yaml", program: "rng.java", drawn: "730 240 637"},
	{runner: "dotnet.yaml", program: "rng.cs", drawn: "358 105 675"},
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
	directory := repositoryPath(t, "runners")
	runner, err := script.ParseRunnerFromFile(filepath.Join(directory, name))
	assert.NoError(t, err)

	if _, err := exec.LookPath(runner.Executable); err != nil {
		if os.Getenv("PRESCRIPT_REQUIRE_RUNNERS") != "" {
			t.Fatalf("%s is not installed, so %s cannot be checked, and PRESCRIPT_REQUIRE_RUNNERS says it has to be", runner.Executable, name)
		}
		t.Skipf("%s is not installed, so %s cannot be checked here", runner.Executable, name)
	}

	if shim := missingShim(runner, directory); shim != "" {
		if os.Getenv("PRESCRIPT_REQUIRE_RUNNERS") != "" {
			t.Fatalf("%s does not exist, so %s cannot be checked, and PRESCRIPT_REQUIRE_RUNNERS says it has to be; run make shims", shim, name)
		}
		t.Skipf("%s does not exist, so %s cannot be checked here; run make shims", shim, name)
	}

	return runner
}

// missingShim names a file the runner points at that is not there.
//
// The Java shim is built rather than checked in, and a runner whose jar is
// missing fails the same way a broken shim would: the program runs, draws its
// own numbers and disagrees with the script. That is a confusing way to be
// told to run make shims.
//
// Written as a search for the runner's own directory rather than a check of
// whole values, because a shim is always named inside something else --
// -javaagent:<jar>, -r<file>, --property:Name=<file>.
func missingShim(runner script.Runner, directory string) string {
	values := append([]string{}, runner.Arguments...)
	for _, value := range runner.Env {
		values = append(values, value)
	}

	for _, value := range values {
		start := strings.Index(value, directory)
		if start < 0 {
			continue
		}

		path := strings.FieldsFunc(value[start:], func(r rune) bool { return r == ' ' || r == '"' })[0]
		if _, err := os.Stat(path); err != nil {
			return path
		}
	}

	return ""
}

// withoutShims keeps the arguments that are only how a language is started --
// `go run`, `dotnet run` -- and drops the ones that activate a shim.
//
// Both kinds live in the same list, so dropping the list wholesale would be
// testing that `go` with no subcommand fails, which it does for reasons that
// have nothing to do with randomness. The rule is the same one missingShim
// uses: an argument that names something in the runner's own directory is the
// runner doing its job, and everything else is just the command line.
func withoutShims(runner script.Runner, directory string) []string {
	var kept []string
	for _, argument := range runner.Arguments {
		if !strings.Contains(argument, directory) {
			kept = append(kept, argument)
		}
	}
	return kept
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
			run.RunnerArguments = withoutShims(runner, repositoryPath(t, "runners"))

			// TERM comes along for the same reason `dotnet run` does: it is
			// how the language is started rather than how it is seeded. Left
			// out, .NET's console announces itself in escape codes ahead of
			// the program's first prompt, and this test fails a step earlier
			// than it means to, for a reason that is not randomness.
			if term, set := runner.Env["TERM"]; set {
				run.InheritEnv = true
				run.Env = map[string]string{"TERM": term}
			}

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
	{runner: "java.yaml", program: "rng.java"},
	{runner: "dotnet.yaml", program: "rng.cs"},
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

// Nobody has to ask. The classification is only useful if it happens on every
// run, and a count that appears when somebody remembered to set a variable is
// one that mostly does not appear.
func TestPrescriptCollectsTheDrawCountItself(t *testing.T) {
	for _, language := range tapedRunners {
		t.Run(language.runner, func(t *testing.T) {
			t.Parallel()
			runner := loadRunner(t, language.runner)
			runs := script.ApplyRunner([]script.Run{drawingRun(t, language.program, referenceDraws)}, runner)

			outcome := Run(tapedConfig(t), runs[0], &utils.CustomLogger{})

			assert.Equal(t, 0, outcome.ExitCode)
			if assert.NotNil(t, outcome.Draws) {
				assert.Equal(t, 5, *outcome.Draws)
			}
		})
	}
}

// Unknown is not zero. Without a tape there is no shim keeping count, and a
// run that cannot say what it drew must not be read as one that drew nothing.
func TestWithoutATapeTheDrawCountIsUnknown(t *testing.T) {
	for _, language := range seededRunners {
		t.Run(language.runner, func(t *testing.T) {
			t.Parallel()
			runner := loadRunner(t, language.runner)
			runs := script.ApplyRunner([]script.Run{drawingRun(t, language.program, language.drawn)}, runner)

			outcome := Run(drawingConfig(), runs[0], &utils.CustomLogger{})

			assert.Equal(t, 0, outcome.ExitCode)
			assert.Nil(t, outcome.Draws)
		})
	}
}

// A run that is killed for hanging runs no exit handler, so a count written
// only on the way out would be missing from exactly the divergences that need
// classifying. The shims write it as they go.
func TestTheDrawCountSurvivesARunThatIsKilled(t *testing.T) {
	for _, language := range tapedRunners {
		t.Run(language.runner, func(t *testing.T) {
			t.Parallel()
			runner := loadRunner(t, language.runner)

			// One step short of what the program prints, so prescript is still
			// waiting when the timeout kills it.
			run := drawingRun(t, language.program, referenceDraws)
			run.Steps = append(run.Steps, script.Step{Line: "never printed"})
			runs := script.ApplyRunner([]script.Run{run}, runner)

			config := tapedConfig(t)
			config.Timeout = getTimeout(2000)

			outcome := Run(config, runs[0], &utils.CustomLogger{})

			assert.NotEqual(t, 0, outcome.ExitCode)
			if assert.NotNil(t, outcome.Draws) {
				assert.Equal(t, 5, *outcome.Draws)
			}
		})
	}
}

// The report end to end, against a language that is actually installed: the
// facts a unit test has to stub -- what the interpreter says it is, how much
// of the tape the run drew, what the tape says about where it came from -- are
// exactly the ones that make a report answerable, so at least once they are
// gathered from a real run rather than described.
func TestABugReportGathersTheFactsFromARealRun(t *testing.T) {
	t.Parallel()
	runner := loadRunner(t, "ruby.yaml")

	// The reference's own draws, minus one, so the run diverges on the last
	// line having drawn the whole tape's worth first.
	wrong := strings.Replace(referenceDraws, "973", "972", 1)
	runs := script.ApplyRunner([]script.Run{drawingRun(t, "rng.rb", wrong)}, runner)

	config := tapedConfig(t)
	config.ScriptFile = "rng.yaml"
	config.BugReport = filepath.Join(t.TempDir(), "report.md")

	outcome := Run(config, runs[0], &utils.CustomLogger{})
	assert.NotEqual(t, 0, outcome.ExitCode)

	written, err := WriteBugReport(config, runs, []Outcome{outcome},
		[]string{"/somewhere/prescript", "play", "rng.yaml", "--tape", tapePath(t)})
	assert.NoError(t, err)
	assert.True(t, written)

	contents, err := os.ReadFile(config.BugReport)
	assert.NoError(t, err)
	report := string(contents)

	assert.Contains(t, report, "| ruby | `ruby ")
	assert.Contains(t, report, "| draws | `5` |")
	assert.Contains(t, report, "# source: vintage-basic")
	assert.Contains(t, report, "prescript play rng.yaml --tape "+tapePath(t))
	assert.Contains(t, report, referenceDraws)
}
