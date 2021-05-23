package script

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBasicScript(t *testing.T) {
	basicScript := `{
  "version": "0.1",
  "runs": [{
    "timestamp": "2021-04-08T23:21:42Z",
	"executable": "/bin/ls",
    "arguments": [
      "-l"
    ],
    "exitCode": 0,
    "steps": [{
      "line": "enter your name: ",
      "input": "richard"
    }, {
      "line": "hello \\w+",
      "isRegex": true
    }]
  }]
}`
	script, err := NewScriptFromBytes([]byte(basicScript))
	assert.NoError(t, err)
	assert.Equal(t, "0.1", script.Version)
	assert.Len(t, script.Runs, 1)

	run := script.Runs[0]
	assert.Equal(t, "2021-04-08T23:21:42Z", run.Timestamp.Format(time.RFC3339))

	assert.Equal(t, "/bin/ls", run.Executable)
	assert.Equal(t, []string{"-l"}, run.Arguments)
	assert.Equal(t, 0, run.ExitCode)

	assert.Len(t, run.Steps, 2)

	assert.Equal(t, "enter your name: ", run.Steps[0].Line)
	assert.Equal(t, "richard", run.Steps[0].Input)
	assert.Equal(t, false, run.Steps[0].IsRegex)

	assert.Equal(t, "hello \\w+", run.Steps[1].Line)
	assert.Equal(t, "", run.Steps[1].Input)
	assert.Equal(t, true, run.Steps[1].IsRegex)
}

func TestValidationFailsForMissingProperties(t *testing.T) {
	basicScript := `{
  "version": "0.1",
}`
	_, err := NewScriptFromBytes([]byte(basicScript))
	assert.Error(t, err)
}

func TestValidationFailsForIncorrectType(t *testing.T) {
	basicScript := `{
  "version": 0,
  "runs": [{
    "timestamp": "2021-04-23:21:42Z",
	"cmd": "ls",
    "arguments": [
      "-l"
    ],
    "exitCode": "zero",
    "steps": [{
      "line": "enter your name: ",
      "input": "richard"
    }, {
      "line": "hello \\w+",
      "isRegex": true
    }]
  }]
}`
	_, err := NewScriptFromBytes([]byte(basicScript))
	assert.Error(t, err)
	expected := `Script validation errors:
runs.0.exitCode: Invalid type. Expected: number, given: string
runs.0.timestamp: Does not match format 'date-time'
runs.0: Additional property cmd is not allowed
version: Invalid type. Expected: string, given: integer`
	assert.Equal(t, expected, err.Error())
}

func TestValidationFailsForIncorrectVersion(t *testing.T) {
	basicScript := `{
  "version": "0.0",
  "runs": []
  }]
}`
	_, err := NewScriptFromBytes([]byte(basicScript))
	assert.Error(t, err)
	expected := `Script validation errors:
runs: Array must have at least 1 items
version: version must be one of the following: "0.1"`
	assert.Equal(t, expected, err.Error())
}

func TestHandlesInvalidRegex(t *testing.T) {
	basicScript := `{
  "version": "0.1",
  "runs": [{
    "arguments": [
      "-l"
    ],
    "exitCode": 0,
    "steps": [{
      "line": "hello (w+",
      "isRegex": true
    }]
  }]
}`
	_, err := NewScriptFromBytes([]byte(basicScript))
	assert.Error(t, err)
	expected := `Script validation errors:
runs.0.steps.0.line: error parsing regexp: missing closing ): ` + "`hello (w+`"
	assert.Equal(t, expected, err.Error())
}
