package lib

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

	fullPath, err := filepath.Abs(filepath.Join(cwd, "..", fileName))
	if err != nil {
		t.Errorf("failed to get absolute path for fixture: %s", err.Error())
	}

	return fullPath
}

func createParams(t *testing.T, fileName string) RunParameters {
	fixture := getFixturePath(fileName, t)
	return RunParameters{
		AppFilePath:           fixture,
		Logger:                zap.NewNop(),
		TimeoutInMilliseconds: 5000,
	}
}

func TestOutput(t *testing.T) {
	t.Parallel()
	params := createParams(t, "fixtures/output.sh")
	exitCode := RunPlay(params)
	assert.Equal(t, 0, exitCode)
}

func TestOutputWithDelay(t *testing.T) {
	t.Parallel()
	params := createParams(t, "fixtures/output_with_delay.sh")
	exitCode := RunPlay(params)
	assert.Equal(t, 0, exitCode)
}

func TestInput(t *testing.T) {
	t.Parallel()

	params := createParams(t, "fixtures/input.sh")
	step := Step{Line: "Please enter your name: ", Input: "Richard"}
	params.Steps = []Step{step}

	exitCode := RunPlay(params)

	assert.Equal(t, 0, exitCode)
}

func TestDoubleInput(t *testing.T) {
	t.Parallel()

	params := createParams(t, "fixtures/double_input.sh")
	lineOne := Step{Line: "First number: ", Input: "1"}
	lineTwo := Step{Line: "Second number: ", Input: "2"}
	params.Steps = []Step{lineOne, lineTwo}

	exitCode := RunPlay(params)

	assert.Equal(t, 0, exitCode)
}

func TestDoubleInputRegex(t *testing.T) {
	t.Parallel()

	params := createParams(t, "fixtures/double_input.sh")
	lineOne := Step{Line: "First number: ", Input: "1"}
	lineTwo := Step{Line: "Second number: ", Input: "2"}
	lineThree := Step{Line: "Sum: \\d", IsRegex: true}
	params.Steps = []Step{lineOne, lineTwo, lineThree}

	exitCode := RunPlay(params)

	assert.Equal(t, 0, exitCode)
}

func TestFailIfUnrecognisedStep(t *testing.T) {
	t.Parallel()

	params := createParams(t, "fixtures/output.sh")
	output := Step{Line: "Hello, Rachel"}
	params.Steps = []Step{output}
	params.TimeoutInMilliseconds = 1000

	exitCode := RunPlay(params)

	assert.Equal(t, 1, exitCode)
}

func TestFailIfUnexpectedStdin(t *testing.T) {
	t.Parallel()

	params := createParams(t, "fixtures/input.sh")
	output := Step{Line: "Hello, Rachel"}
	params.Steps = []Step{output}
	params.TimeoutInMilliseconds = 1000

	exitCode := RunPlay(params)

	assert.Equal(t, 1, exitCode)
}
