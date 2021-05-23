package play

import (
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
	exitCode := Run(config.Play, script.Run{}, config.Logger)
	assert.Equal(t, 0, exitCode)
}

func TestOutputWithDelay(t *testing.T) {
	t.Parallel()
	config := createConfig(t, "fixtures/output_with_delay.sh")
	exitCode := Run(config.Play, script.Run{}, config.Logger)
	assert.Equal(t, 0, exitCode)
}

func TestInput(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/input.sh")
	run := script.Run{Steps: []script.Step{{Line: "Please enter your name: ", Input: "Richard"}}}

	exitCode := Run(config.Play, run, config.Logger)

	assert.Equal(t, 0, exitCode)
}

func TestDoubleInput(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/double_input.sh")
	run := script.Run{Steps: []script.Step{
		{Line: "First number: ", Input: "1"},
		{Line: "Second number: ", Input: "2"},
	}}

	exitCode := Run(config.Play, run, config.Logger)

	assert.Equal(t, 0, exitCode)
}

func TestDoubleInputRegex(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/double_input.sh")
	regex := regexp.MustCompile(`Sum: \d`)
	run := script.Run{Steps: []script.Step{
		{Line: "First number: ", Input: "1"},
		{Line: "Second number: ", Input: "2"},
		{Line: "Sum: \\d", LineRegex: *regex, IsRegex: true},
	}}

	exitCode := Run(config.Play, run, config.Logger)

	assert.Equal(t, 0, exitCode)
}

func TestPassingArguments(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/input_arguments.sh")
	run := script.Run{
		Steps:     []script.Step{{Line: "Hello, Rachel"}},
		Arguments: []string{"Rachel"},
	}

	exitCode := Run(config.Play, run, config.Logger)

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

	exitCode := Run(config.Play, run, config.Logger)

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
	exitCode := Run(config.Play, run, config.Logger)

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
	exitCode := Run(config.Play, run, config.Logger)

	assert.Equal(t, 0, exitCode)
}

func TestFailIfUnrecognisedStep(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/output.sh")
	config.Play.Timeout = getTimeout(1000)
	run := script.Run{Steps: []script.Step{
		{Line: "Hello, Rachel"},
	}}

	exitCode := Run(config.Play, run, config.Logger)

	assert.Equal(t, 1, exitCode)
}

func TestFailIfUnexpectedStdin(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/input.sh")
	config.Play.Timeout = getTimeout(1000)
	run := script.Run{Steps: []script.Step{
		{Line: "Hello, Rachel"},
	}}

	exitCode := Run(config.Play, run, config.Logger)

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

	exitCode := Run(config.Play, run, config.Logger)

	assert.Equal(t, 1, exitCode)
}
