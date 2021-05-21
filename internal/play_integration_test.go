package internal

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	cfg "prescript/internal/config"
)

func getFixturePath(fileName string, t *testing.T) string {
	cwd, err := os.Getwd()
	if err != nil {
		t.Errorf("failed to get current working directory: %s", err.Error())
	}

	fullPath, err := filepath.Abs(filepath.Join(cwd, "..", "test", fileName))
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
		Logger: &CustomLogger{},
	}
}

func TestOutput(t *testing.T) {
	t.Parallel()
	config := createConfig(t, "fixtures/output.sh")
	exitCode := RunPlay(config, Run{})
	assert.Equal(t, 0, exitCode)
}

func TestOutputWithDelay(t *testing.T) {
	t.Parallel()
	config := createConfig(t, "fixtures/output_with_delay.sh")
	exitCode := RunPlay(config, Run{})
	assert.Equal(t, 0, exitCode)
}

func TestInput(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/input.sh")
	run := Run{Steps: []Step{{Line: "Please enter your name: ", Input: "Richard"}}}

	exitCode := RunPlay(config, run)

	assert.Equal(t, 0, exitCode)
}

func TestDoubleInput(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/double_input.sh")
	run := Run{Steps: []Step{
		{Line: "First number: ", Input: "1"},
		{Line: "Second number: ", Input: "2"},
	}}

	exitCode := RunPlay(config, run)

	assert.Equal(t, 0, exitCode)
}

func TestDoubleInputRegex(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/double_input.sh")
	regex := regexp.MustCompile(`Sum: \d`)
	run := Run{Steps: []Step{
		{Line: "First number: ", Input: "1"},
		{Line: "Second number: ", Input: "2"},
		{Line: "Sum: \\d", LineRegex: *regex, IsRegex: true},
	}}

	exitCode := RunPlay(config, run)

	assert.Equal(t, 0, exitCode)
}

func TestPassingArguments(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/input_arguments.sh")
	run := Run{
		Steps:     []Step{{Line: "Hello, Rachel"}},
		Arguments: []string{"Rachel"},
	}

	exitCode := RunPlay(config, run)

	assert.Equal(t, 0, exitCode)
}

func TestSpecifyExecutableInScript(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/input_arguments.sh")
	run := Run{
		Executable: config.Play.ExecutablePath,
		Steps:      []Step{{Line: "Hello, Rachel"}},
		Arguments:  []string{"Rachel"},
	}
	config.Play.ExecutablePath = ""

	exitCode := RunPlay(config, run)

	assert.Equal(t, 0, exitCode)
}

func TestDuplicatedLinesInScriptt(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/duplicated_lines.sh")
	run := Run{
		Executable: config.Play.ExecutablePath,
		Steps: []Step{
			{Line: "the same line repeated"},
			{Line: "the same line repeated"},
			{Line: "Please enter your name: ", Input: "Harold"},
			{Line: "Your name is Harold"},
			{Line: "more lines repeated"},
			{Line: "more lines repeated"},
			{Line: "success!"},
		},
	}
	exitCode := RunPlay(config, run)

	assert.Equal(t, 0, exitCode)
}

func TestFailIfUnrecognisedStep(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/output.sh")
	config.Play.Timeout = getTimeout(1000)
	run := Run{Steps: []Step{
		{Line: "Hello, Rachel"},
	}}

	exitCode := RunPlay(config, run)

	assert.Equal(t, 1, exitCode)
}

func TestFailIfUnexpectedStdin(t *testing.T) {
	t.Parallel()

	config := createConfig(t, "fixtures/input.sh")
	config.Play.Timeout = getTimeout(1000)
	run := Run{Steps: []Step{
		{Line: "Hello, Rachel"},
	}}

	exitCode := RunPlay(config, run)

	assert.Equal(t, 1, exitCode)
}
