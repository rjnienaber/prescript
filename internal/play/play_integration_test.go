package play

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	cfg "github.com/rjnienaber/prescript/internal/config"
	"github.com/rjnienaber/prescript/internal/script"
	"github.com/rjnienaber/prescript/internal/utils"
	"github.com/stretchr/testify/assert"
)

func getFixturePath(fileName string, t *testing.T) string {
	cwd, err := os.Getwd()
	if err != nil {
		t.Errorf("failed to get current working directory: %s", err.Error())
	}

	fullPath, err := filepath.Abs(filepath.Join(cwd, "..", "..", "test", fileName))
	if err != nil {
		t.Errorf("failed to get absolute path for fixture: %s", err.Error())
	}

	return fullPath
}

func getTimeout(timeoutInMilliseconds int) time.Duration {
	return time.Duration(timeoutInMilliseconds) * time.Millisecond
}

func createConfig(t *testing.T, fileName string) cfg.Config {
	executablePath := getFixturePath(fileName, t)
	return cfg.Config{
		Subcommand: cfg.PlayCommand,
		Play: cfg.PlayConfig{
			Timeout:        getTimeout(5000),
			ExecutablePath: executablePath,
		},
		Logger: &utils.CustomLogger{},
	}
}

func TestOutput(t *testing.T) {
	t.Parallel()
	config := createConfig(t, "fixtures/output.sh")
	exitCode := Run(config.Play, script.Run{}, config.Logger).ExitCode
	assert.Equal(t, 0, exitCode)
}

func TestOutputWithDelay(t *testing.T) {
	t.Parallel()
	config := createConfig(t, "fixtures/output_with_delay.sh")
	exitCode := Run(config.Play, script.Run{}, config.Logger).ExitCode
	assert.Equal(t, 0, exitCode)
}

func TestInput(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/input.sh")
	run := script.Run{Steps: []script.Step{{Line: "Please enter your name: ", Input: "Richard"}}}

	exitCode := Run(config.Play, run, config.Logger).ExitCode

	assert.Equal(t, 0, exitCode)
}

func TestDoubleInput(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/double_input.sh")
	run := script.Run{Steps: []script.Step{
		{Line: "First number: ", Input: "1"},
		{Line: "Second number: ", Input: "2"},
	}}

	exitCode := Run(config.Play, run, config.Logger).ExitCode

	assert.Equal(t, 0, exitCode)
}

func TestDoubleInputRedaction(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/double_input.sh")
	regex := regexp.MustCompile(`Sum: \d`)
	run := script.Run{Steps: []script.Step{
		{Line: "First number: ", Input: "1"},
		{Line: "Second number: ", Input: "2"},
		{Line: "Sum: {{digit}}", LineRegex: *regex, Redacted: true},
	}}

	exitCode := Run(config.Play, run, config.Logger).ExitCode

	assert.Equal(t, 0, exitCode)
}

func TestPassingArguments(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/input_arguments.sh")
	run := script.Run{
		Steps:     []script.Step{{Line: "Hello, Rachel"}},
		Arguments: []string{"Rachel"},
	}

	exitCode := Run(config.Play, run, config.Logger).ExitCode

	assert.Equal(t, 0, exitCode)
}

func TestSpecifyExecutableInScript(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/input_arguments.sh")
	run := script.Run{
		Executable: config.Play.ExecutablePath,
		Steps:      []script.Step{{Line: "Hello, Rachel"}},
		Arguments:  []string{"Rachel"},
	}
	config.Play.ExecutablePath = ""

	exitCode := Run(config.Play, run, config.Logger).ExitCode

	assert.Equal(t, 0, exitCode)
}

func TestDuplicatedLinesInScript(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/duplicated_lines.sh")
	run := script.Run{
		Executable: config.Play.ExecutablePath,
		Steps: []script.Step{
			{Line: "the same line repeated"},
			{Line: "the same line repeated"},
			{Line: "Please enter your name: ", Input: "Harold"},
			{Line: "Your name is Harold"},
			{Line: "more lines repeated"},
			{Line: "more lines repeated"},
			{Line: "success!"},
		},
	}
	exitCode := Run(config.Play, run, config.Logger).ExitCode

	assert.Equal(t, 0, exitCode)
}

func TestCheckExitCode(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/exit_code.sh")
	run := script.Run{
		Executable: config.Play.ExecutablePath,
		Steps: []script.Step{
			{Line: "Exit Code test"},
		},
		ExitCode: 1,
	}
	exitCode := Run(config.Play, run, config.Logger).ExitCode

	assert.Equal(t, 0, exitCode)
}

func TestFailIfUnrecognisedStep(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/output.sh")
	config.Play.Timeout = getTimeout(1000)
	run := script.Run{Steps: []script.Step{
		{Line: "Hello, Rachel"},
	}}

	exitCode := Run(config.Play, run, config.Logger).ExitCode

	assert.Equal(t, 1, exitCode)
}

func TestFailIfUnexpectedStdin(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/input.sh")
	config.Play.Timeout = getTimeout(1000)
	run := script.Run{Steps: []script.Step{
		{Line: "Hello, Rachel"},
	}}

	exitCode := Run(config.Play, run, config.Logger).ExitCode

	assert.Equal(t, 1, exitCode)
}

func TestFailIfExecutableTimesOutAfterSteps(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/timeout.sh")
	logger, _ := utils.NewLogger("debug")
	config.Play.Timeout = getTimeout(1000)
	config.Logger = &logger
	run := script.Run{Steps: []script.Step{
		{Line: "Expecting this line"},
	}}

	exitCode := Run(config.Play, run, config.Logger).ExitCode

	assert.Equal(t, 1, exitCode)
}

// The three tests below are not parallel: t.Setenv cannot be used in a parallel
// test, and the point of each is what the child sees of prescript's own
// environment.

func TestDeclaredEnvReplacesTheParentEnvironment(t *testing.T) {
	t.Setenv("PRESCRIPT_TEST_PARENT", "visible")

	config := createConfig(t, "fixtures/env.sh")
	run := script.Run{
		Env: map[string]string{"GREETING": "hello"},
		Steps: []script.Step{
			{Line: "GREETING=hello"},
			{Line: "PARENT=unset"},
		},
	}

	exitCode := Run(config.Play, run, config.Logger).ExitCode

	assert.Equal(t, 0, exitCode)
}

func TestInheritEnvKeepsTheParentEnvironment(t *testing.T) {
	t.Setenv("PRESCRIPT_TEST_PARENT", "visible")

	config := createConfig(t, "fixtures/env.sh")
	run := script.Run{
		Env:        map[string]string{"GREETING": "hello"},
		InheritEnv: true,
		Steps: []script.Step{
			{Line: "GREETING=hello"},
			{Line: "PARENT=visible"},
		},
	}

	exitCode := Run(config.Play, run, config.Logger).ExitCode

	assert.Equal(t, 0, exitCode)
}

func TestNoDeclaredEnvInheritsTheParentEnvironment(t *testing.T) {
	t.Setenv("PRESCRIPT_TEST_PARENT", "visible")

	config := createConfig(t, "fixtures/env.sh")
	run := script.Run{Steps: []script.Step{
		{Line: "GREETING=unset"},
		{Line: "PARENT=visible"},
	}}

	exitCode := Run(config.Play, run, config.Logger).ExitCode

	assert.Equal(t, 0, exitCode)
}

// End to end through the real parser: a script whose expected line covers a
// value that is different on every run, which is the case redactions exist for
// and the case a literal line cannot express at all.
func TestRedactionMatchesAVolatileValue(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/volatile.sh")
	document := `{
  "version": "0.4",
  "redactions": {"pid": "[0-9]+"},
  "runs": [{"arguments": [], "exitCode": 0, "steps": [{"line": "PROCESS {{pid}} STARTED"}]}]
}`
	parsed, err := script.ParseScriptFromBytes([]byte(document))
	assert.NoError(t, err)

	exitCode := Run(config.Play, parsed.Runs[0], config.Logger).ExitCode

	assert.Equal(t, 0, exitCode)
}

func TestRedactionStillFailsOnARealMismatch(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/volatile.sh")
	document := `{
  "version": "0.4",
  "redactions": {"pid": "[0-9]+"},
  "runs": [{"arguments": [], "exitCode": 0, "steps": [{"line": "THREAD {{pid}} STARTED"}]}]
}`
	parsed, err := script.ParseScriptFromBytes([]byte(document))
	assert.NoError(t, err)

	exitCode := Run(config.Play, parsed.Runs[0], config.Logger).ExitCode

	assert.NotEqual(t, 0, exitCode)
}

// What a pty is for: a program that asks whether it is talking to a terminal
// gets a different answer, and the answer is the one it would get from a
// person at a keyboard.
func TestExecutableRunsUnderAPtyByDefault(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/tty.sh")
	run := script.Run{Steps: []script.Step{
		{Line: "STDOUT=terminal"},
		{Line: "Name: ", Input: "Rachel"},
		{Line: "Hello, Rachel"},
	}}

	exitCode := Run(config.Play, run, config.Logger).ExitCode

	assert.Equal(t, 0, exitCode)
}

// A program asking how wide its terminal is gets the same answer here as it
// does on the next machine, whatever terminal prescript itself was started
// from -- and an answer at all, which a pty opened without a size does not
// give: that one reports zero rows and columns.
func TestThePtyIsAlwaysTheSameSize(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/terminal_size.sh")
	run := script.Run{Steps: []script.Step{{Line: "24 80"}}}

	exitCode := Run(config.Play, run, config.Logger).ExitCode

	assert.Equal(t, 0, exitCode)
}

func TestTerminalPipesGivesTheExecutablePipes(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/tty.sh")
	config.Play.Terminal = "pipes"
	run := script.Run{Steps: []script.Step{
		{Line: "STDOUT=pipe"},
		{Line: "Name: ", Input: "Rachel"},
		{Line: "Hello, Rachel"},
	}}

	exitCode := Run(config.Play, run, config.Logger).ExitCode

	assert.Equal(t, 0, exitCode)
}

func TestRunCanAskForPipes(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/tty.sh")
	run := script.Run{
		Terminal: "pipes",
		Steps: []script.Step{
			{Line: "STDOUT=pipe"},
			{Line: "Name: ", Input: "Rachel"},
			{Line: "Hello, Rachel"},
		},
	}

	exitCode := Run(config.Play, run, config.Logger).ExitCode

	assert.Equal(t, 0, exitCode)
}

// --terminal is the way to try a script the other way round without editing
// it, so it has to beat what the script declared.
func TestCommandLineTerminalOverridesTheScript(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/tty.sh")
	config.Play.Terminal = "pty"
	run := script.Run{
		Terminal: "pipes",
		Steps: []script.Step{
			{Line: "STDOUT=terminal"},
			{Line: "Name: ", Input: "Rachel"},
			{Line: "Hello, Rachel"},
		},
	}

	exitCode := Run(config.Play, run, config.Logger).ExitCode

	assert.Equal(t, 0, exitCode)
}

func TestUnrecognisedTerminalIsRejected(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/tty.sh")
	config.Play.Terminal = "tty"

	exitCode := Run(config.Play, script.Run{}, config.Logger).ExitCode

	assert.Equal(t, utils.USER_ERROR, exitCode)
}

// Echo is what makes a pty different from a pipe for input, and it has to be
// off: with it on, everything prescript types would come straight back as
// output the next step would have to expect.
func TestInputIsNotEchoedBackUnderAPty(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/input.sh")
	run := script.Run{Steps: []script.Step{
		{Line: "Please enter your name: ", Input: "Richard"},
		// Under an echoing terminal this line arrives prefixed with the
		// "Richard" prescript just typed, and never matches.
		{Line: "How do you do, Richard"},
	}}

	exitCode := Run(config.Play, run, config.Logger).ExitCode

	assert.Equal(t, 0, exitCode)
}

// captureStderr collects a run's failure report. It swaps os.Stderr, so a test
// using it cannot be parallel.
func captureStderr(t *testing.T, play func()) string {
	t.Helper()

	original := os.Stderr
	reader, writer, err := os.Pipe()
	assert.NoError(t, err)
	os.Stderr = writer

	captured := make(chan string, 1)
	go func() {
		var buffer bytes.Buffer
		_, _ = io.Copy(&buffer, reader)
		captured <- buffer.String()
	}()

	play()

	os.Stderr = original
	assert.NoError(t, writer.Close())
	return <-captured
}

// The three ways a run ends short are three different bugs, and the report
// says which one it was before it says anything else.
func TestFailureModesAreNamed(t *testing.T) {
	// The program is still running and printing something else.
	report := captureStderr(t, func() {
		config := createConfig(t, "fixtures/input.sh")
		config.Play.Timeout = getTimeout(500)
		run := script.Run{Steps: []script.Step{{Line: "Hello, Rachel"}}}
		assert.Equal(t, utils.CLI_ERROR, Run(config.Play, run, config.Logger).ExitCode)
	})
	assert.Contains(t, report, "no-match: step 1 of 1 did not match (nothing matched it within 500ms)")

	// The program finished with steps still outstanding.
	report = captureStderr(t, func() {
		config := createConfig(t, "fixtures/output.sh")
		run := script.Run{Steps: []script.Step{{Line: "Hello, Rachel"}}}
		assert.Equal(t, utils.CLI_ERROR, Run(config.Play, run, config.Logger).ExitCode)
	})
	assert.Contains(t, report, "exited-early: step 1 of 1 did not match (the executable exited with 0)")

	// Every step matched and the program never exited.
	report = captureStderr(t, func() {
		config := createConfig(t, "fixtures/timeout.sh")
		config.Play.Timeout = getTimeout(500)
		run := script.Run{Steps: []script.Step{{Line: "Expecting this line"}}}
		assert.Equal(t, utils.CLI_ERROR, Run(config.Play, run, config.Logger).ExitCode)
	})
	assert.Equal(t, "hung: all 1 steps matched, but the executable had not exited after 500ms\n", report)

	// It printed everything expected of it and disagreed only about the end.
	report = captureStderr(t, func() {
		config := createConfig(t, "fixtures/exit_code.sh")
		run := script.Run{Steps: []script.Step{{Line: "Exit Code test"}}}
		assert.Equal(t, utils.INTERNAL_ERROR, Run(config.Play, run, config.Logger).ExitCode)
	})
	assert.Contains(t, report, "wrong-exit-code: all 1 steps matched, but the executable exited with 1 and the script expects 0")
}

// A step that knows it is slow says so, rather than every other step in the
// script being given the same patience.
func TestStepTimeoutOverridesTheRunTimeout(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/slow_step.sh")
	config.Play.Timeout = getTimeout(500)
	run := script.Run{Steps: []script.Step{
		{Line: "Working"},
		{Line: "Done", Timeout: "5s", TimeoutDuration: 5 * time.Second},
	}}

	exitCode := Run(config.Play, run, config.Logger).ExitCode

	assert.Equal(t, 0, exitCode)
}

// The same script without the override, so the test above is known to be
// proving the override rather than a generous default.
func TestSlowStepFailsWithoutAnOverride(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/slow_step.sh")
	config.Play.Timeout = getTimeout(500)
	run := script.Run{Steps: []script.Step{
		{Line: "Working"},
		{Line: "Done"},
	}}

	exitCode := Run(config.Play, run, config.Logger).ExitCode

	assert.Equal(t, utils.CLI_ERROR, exitCode)
}

// A run that gives up on a program has to take it, and whatever it started,
// with it. In a corpus run the alternative is one leaked process per timeout.
func TestTimeoutKillsTheExecutableAndItsChildren(t *testing.T) {
	t.Parallel()

	marker := filepath.Join(t.TempDir(), "alive")
	config := createConfig(t, "fixtures/hangs_with_child.sh")
	config.Play.Timeout = getTimeout(300)
	run := script.Run{
		Arguments: []string{marker},
		Steps:     []script.Step{{Line: "Started"}},
	}

	assert.Equal(t, utils.CLI_ERROR, Run(config.Play, run, config.Logger).ExitCode)

	// The background process writes the marker a second in. Long enough after
	// that to be sure: it either never ran again, or it did.
	time.Sleep(1500 * time.Millisecond)
	_, err := os.Stat(marker)
	assert.True(t, os.IsNotExist(err), "the executable's background process outlived the run")
}
