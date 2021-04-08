package main

import (
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"prescript/lib"
	"testing"
)

func getFixturePath(fileName string, t *testing.T) string {
	cwd, err := os.Getwd()
	if err != nil {
		t.Errorf("failed to get current working directory: %s", err.Error())
	}

	fullPath, err := filepath.Abs(filepath.Join(cwd, fileName))
	if err != nil {
		t.Errorf("failed to get absolute path for fixture: %s", err.Error())
	}

	return fullPath
}

func createParams(t *testing.T, fileName string) lib.RunParameters {
	fixture := getFixturePath(fileName, t)
	return lib.RunParameters{
		AppFilePath:           fixture,
		Logger:                zap.NewNop(),
		TimeoutInMilliseconds: 5000,
	}
}

func TestOutput(t *testing.T) {
	t.Parallel()
	params := createParams(t, "fixtures/output.sh")
	exitCode := Run(params)
	assert.Equal(t, 0, exitCode)
}

func TestOutputWithDelay(t *testing.T) {
	t.Parallel()
	params := createParams(t, "fixtures/output_with_delay.sh")
	exitCode := Run(params)
	assert.Equal(t, 0, exitCode)
}

func TestInput(t *testing.T) {
	t.Parallel()

	params := createParams(t, "fixtures/input.sh")
	step := lib.Step{Line: "Please enter your name: ", Input: "Richard"}
	params.Steps = []lib.Step{step}

	exitCode := Run(params)

	assert.Equal(t, 0, exitCode)
}

func TestDoubleInput(t *testing.T) {
	t.Parallel()

	params := createParams(t, "fixtures/double_input.sh")
	lineOne := lib.Step{Line: "First number: ", Input: "1"}
	lineTwo := lib.Step{Line: "Second number: ", Input: "2"}
	params.Steps = []lib.Step{lineOne, lineTwo}

	exitCode := Run(params)

	assert.Equal(t, 0, exitCode)
}

func TestDoubleInputRegex(t *testing.T) {
	t.Parallel()

	params := createParams(t, "fixtures/double_input.sh")
	lineOne := lib.Step{Line: "First number: ", Input: "1"}
	lineTwo := lib.Step{Line: "Second number: ", Input: "2"}
	lineThree := lib.Step{Line: "Sum: \\d", IsRegex: true}
	params.Steps = []lib.Step{lineOne, lineTwo, lineThree}

	exitCode := Run(params)

	assert.Equal(t, 0, exitCode)
}

func TestFailIfUnrecognisedStep(t *testing.T) {
	t.Parallel()

	params := createParams(t, "fixtures/output.sh")
	output := lib.Step{Line: "Hello, Rachel"}
	params.Steps = []lib.Step{output}
	params.TimeoutInMilliseconds = 1000

	exitCode := Run(params)

	assert.Equal(t, 1, exitCode)
}

func TestFailIfUnexpectedStdin(t *testing.T) {
	t.Parallel()

	params := createParams(t, "fixtures/input.sh")
	output := lib.Step{Line: "Hello, Rachel"}
	params.Steps = []lib.Step{output}
	params.TimeoutInMilliseconds = 1000

	exitCode := Run(params)

	assert.Equal(t, 1, exitCode)
}
