package main

import (
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"os"
	"path/filepath"
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

func createParams(t *testing.T, fileName string) RunParameters {
	fixture := getFixturePath(fileName, t)
	return RunParameters{
		appFilePath: fixture,
		logger:      zap.NewNop(),
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
	step := Step{
		line:  "Please enter your name: ",
		input: "Richard",
	}
	params.steps = []Step{step}
	exitCode := Run(params)
	assert.Equal(t, 0, exitCode)
}

func TestDoubleInput(t *testing.T) {
	t.Parallel()
	params := createParams(t, "fixtures/double_input.sh")
	firstInput := Step{
		line:  "First number: ",
		input: "1",
	}
	secondInput := Step{
		line:  "Second number: ",
		input: "2",
	}
	params.steps = []Step{firstInput, secondInput}
	exitCode := Run(params)
	assert.Equal(t, 0, exitCode)
}

func TestFailIfUnexpectedStep(t *testing.T) {
	t.Parallel()
	params := createParams(t, "fixtures/output.sh")
	output := Step{
		line: "Hello, Rachel",
	}
	params.steps = []Step{output}
	params.timeoutInMilliseconds = 1000
	exitCode := Run(params)
	assert.Equal(t, 1, exitCode)
}
