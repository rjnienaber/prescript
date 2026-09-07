package script

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBasicScript(t *testing.T) {
	basicScript := `{
  "version": "0.1",
  "runs": [{
	"name": "list directories",
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
	script, err := ParseScriptFromBytes([]byte(basicScript))
	assert.NoError(t, err)
	assert.Equal(t, "0.1", script.Version)
	assert.Len(t, script.Runs, 1)

	run := script.Runs[0]
	assert.Equal(t, "list directories", run.Name)
	assert.Equal(t, "2021-04-08T23:21:42Z", run.Timestamp.Format(time.RFC3339))

	assert.Equal(t, "/bin/ls", run.Executable)
	assert.Equal(t, []string{"-l"}, run.Arguments)
	assert.Equal(t, 0, run.ExitCode)

	assert.Len(t, run.Steps, 2)

	assert.Equal(t, "enter your name: ", run.Steps[0].Line)
	assert.Equal(t, "richard", run.Steps[0].Input)
	assert.False(t, run.Steps[0].IsRegex)

	stepOne := run.Steps[1]
	assert.Equal(t, "hello \\w+", stepOne.Line)
	assert.Equal(t, "", stepOne.Input)
	assert.True(t, stepOne.IsRegex)
	assert.True(t, stepOne.LineRegex.MatchString("hello richard"))
}

func TestGeneratesNames(t *testing.T) {
	basicScript := `{
  "version": "0.1",
  "runs": [{
    "arguments": [
      "-l"
    ],
    "exitCode": 0,
    "steps": [{
      "line": "hello world"
    }]
  }]
}`
	script, err := ParseScriptFromBytes([]byte(basicScript))
	assert.NoError(t, err)
	assert.Len(t, script.Runs, 1)
	assert.Equal(t, "run 0", script.Runs[0].Name)
}

func TestValidationFailsForMissingProperties(t *testing.T) {
	basicScript := `{
  "version": "0.1",
}`
	_, err := ParseScriptFromBytes([]byte(basicScript))
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
	_, err := ParseScriptFromBytes([]byte(basicScript))
	assert.Error(t, err)
	expected := `Script validation errors:
runs.0.exitCode: Invalid type. Expected: number, given: string
runs.0.timestamp: Does not match format 'date-time'
runs.0: Additional property cmd is not allowed
version: Invalid type. Expected: string, given: integer`
	assert.Equal(t, expected, err.Error())
}

// The expected messages list every version this build reads, so bumping the
// format does not mean rewriting the assertions.
func knownVersionList() string {
	return strings.Join(quoteAll(knownVersions), ", ")
}

func TestValidationFailsForUnrecognisedVersion(t *testing.T) {
	basicScript := `{
  "version": "0.0",
  "runs": []
}`
	_, err := ParseScriptFromBytes([]byte(basicScript))
	assert.Error(t, err)
	// The version is reported on its own: the rest of the document was written
	// against a format this build does not know, so complaining about its shape
	// would be guesswork.
	expected := `Script validation errors:
version: unrecognised script format "0.0": expected one of ` + knownVersionList()
	assert.Equal(t, expected, err.Error())
}

func TestValidationFailsForNewerVersion(t *testing.T) {
	basicScript := `{
  "version": "9.9",
  "runs": [{
    "arguments": [],
    "exitCode": 0,
    "steps": [{"line": "hello", "env": {"A": "B"}}]
  }]
}`
	_, err := ParseScriptFromBytes([]byte(basicScript))
	assert.Error(t, err)
	// Naming the version says what to do about it; a list of unrecognised
	// fields would not.
	expected := `Script validation errors:
version: script is written for format 9.9, but this build of prescript reads up to ` + CurrentVersion + `; upgrade prescript`
	assert.Equal(t, expected, err.Error())
}

func TestValidationFailsForMalformedVersion(t *testing.T) {
	basicScript := `{
  "version": "one",
  "runs": [{"arguments": [], "exitCode": 0, "steps": [{"line": "hello"}]}]
}`
	_, err := ParseScriptFromBytes([]byte(basicScript))
	assert.Error(t, err)
	expected := `Script validation errors:
version: unrecognised script format "one": expected one of ` + knownVersionList()
	assert.Equal(t, expected, err.Error())
}

func TestCurrentVersionIsAccepted(t *testing.T) {
	basicScript := `{
  "version": "` + CurrentVersion + `",
  "runs": [{"arguments": [], "exitCode": 0, "steps": [{"line": "hello"}]}]
}`
	script, err := ParseScriptFromBytes([]byte(basicScript))
	assert.NoError(t, err)
	assert.Equal(t, CurrentVersion, script.Version)
}

// Every version this build claims to read has to be one it can actually parse,
// and has to be well formed, or the comparison in checkVersion is meaningless.
func TestEveryKnownVersionIsReadable(t *testing.T) {
	for _, known := range knownVersions {
		assert.NoError(t, checkVersion(known), known)

		major, minor, err := splitVersion(known)
		assert.NoError(t, err, known)
		assert.GreaterOrEqual(t, major, 0, known)
		assert.GreaterOrEqual(t, minor, 0, known)
	}
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
	_, err := ParseScriptFromBytes([]byte(basicScript))
	assert.Error(t, err)
	expected := `Script validation errors:
runs.0.steps.0.line: error parsing regexp: missing closing ): ` + "`hello (w+`"
	assert.Equal(t, expected, err.Error())
}

func TestParsesEnv(t *testing.T) {
	basicScript := `{
  "version": "0.2",
  "runs": [{
    "arguments": [],
    "exitCode": 0,
    "env": {"VINTBAS_SEED": "0", "TZ": "UTC"},
    "inheritEnv": true,
    "steps": [{"line": "hello"}]
  }]
}`
	script, err := ParseScriptFromBytes([]byte(basicScript))
	assert.NoError(t, err)

	run := script.Runs[0]
	assert.Equal(t, map[string]string{"VINTBAS_SEED": "0", "TZ": "UTC"}, run.Env)
	assert.True(t, run.InheritEnv)
}

func TestValidationFailsForNonStringEnvValue(t *testing.T) {
	basicScript := `{
  "version": "0.2",
  "runs": [{
    "arguments": [],
    "exitCode": 0,
    "env": {"VINTBAS_SEED": 0},
    "steps": [{"line": "hello"}]
  }]
}`
	_, err := ParseScriptFromBytes([]byte(basicScript))
	assert.Error(t, err)
	expected := `Script validation errors:
runs.0.env.VINTBAS_SEED: Invalid type. Expected: string, given: integer`
	assert.Equal(t, expected, err.Error())
}

// A 0.1 script predates env, and has to keep meaning what it meant: inherit.
func TestOlderScriptsStillParse(t *testing.T) {
	basicScript := `{
  "version": "0.1",
  "runs": [{"arguments": [], "exitCode": 0, "steps": [{"line": "hello"}]}]
}`
	script, err := ParseScriptFromBytes([]byte(basicScript))
	assert.NoError(t, err)
	assert.Nil(t, script.Runs[0].Environment([]string{"HOME=/home/richard"}))
}

func TestParsesMultipleRuns(t *testing.T) {
	basicScript := `{
  "version": "0.3",
  "runs": [{
    "name": "reference",
    "executable": "vintbas",
    "arguments": [],
    "exitCode": 0,
    "steps": [{"line": "hello"}]
  }, {
    "name": "port",
    "executable": "python",
    "arguments": [],
    "exitCode": 0,
    "steps": [{"line": "hello"}]
  }]
}`
	script, err := ParseScriptFromBytes([]byte(basicScript))
	assert.NoError(t, err)
	assert.Len(t, script.Runs, 2)
	assert.Equal(t, "reference", script.Runs[0].Name)
	assert.Equal(t, "port", script.Runs[1].Name)
}

func TestValidationFailsForDuplicateRunNames(t *testing.T) {
	basicScript := `{
  "version": "0.3",
  "runs": [
    {"name": "dice", "arguments": [], "exitCode": 0, "steps": [{"line": "hello"}]},
    {"name": "dice", "arguments": [], "exitCode": 0, "steps": [{"line": "hello"}]}
  ]
}`
	_, err := ParseScriptFromBytes([]byte(basicScript))
	assert.Error(t, err)
	expected := `Script validation errors:
runs.1.name: duplicate run name "dice", already used by runs.0`
	assert.Equal(t, expected, err.Error())
}

// A generated name can collide with a declared one, and the check has to see
// both or the ambiguity it exists to catch slips through.
func TestValidationFailsWhenADeclaredNameCollidesWithAGeneratedOne(t *testing.T) {
	basicScript := `{
  "version": "0.3",
  "runs": [
    {"name": "run 1", "arguments": [], "exitCode": 0, "steps": [{"line": "hello"}]},
    {"arguments": [], "exitCode": 0, "steps": [{"line": "hello"}]}
  ]
}`
	_, err := ParseScriptFromBytes([]byte(basicScript))
	assert.Error(t, err)
	expected := `Script validation errors:
runs.1.name: duplicate run name "run 1", already used by runs.0`
	assert.Equal(t, expected, err.Error())
}

func TestUnnamedRunsDoNotCollide(t *testing.T) {
	basicScript := `{
  "version": "0.3",
  "runs": [
    {"arguments": [], "exitCode": 0, "steps": [{"line": "hello"}]},
    {"arguments": [], "exitCode": 0, "steps": [{"line": "hello"}]}
  ]
}`
	script, err := ParseScriptFromBytes([]byte(basicScript))
	assert.NoError(t, err)
	assert.Equal(t, "run 0", script.Runs[0].Name)
	assert.Equal(t, "run 1", script.Runs[1].Name)
}
