package script

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func writeScript(t *testing.T, name string, contents string) string {
	path := filepath.Join(t.TempDir(), name)
	assert.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
	return path
}

const yamlScript = `# A script may be written in YAML, comments and all.
version: "0.3"
runs:
  - name: reference
    executable: vintbas
    arguments: [dice.bas]
    exitCode: 0
    env:
      VINTBAS_SEED: "0"
    steps:
      # Quoted: the trailing space on a prompt is part of what has to match.
      - line: "HOW MANY ROLLS? "
        input: "5000"
      - line: 'TOTAL SPOTS   NUMBER OF TIMES'
`

func TestParsesYamlScript(t *testing.T) {
	script, err := ParseScriptFromFile(writeScript(t, "dice.yaml", yamlScript))
	assert.NoError(t, err)

	run := script.Runs[0]
	assert.Equal(t, "reference", run.Name)
	assert.Equal(t, "vintbas", run.Executable)
	assert.Equal(t, []string{"dice.bas"}, run.Arguments)
	assert.Equal(t, map[string]string{"VINTBAS_SEED": "0"}, run.Env)
	assert.Equal(t, "HOW MANY ROLLS? ", run.Steps[0].Line)
	assert.Equal(t, "5000", run.Steps[0].Input)
}

func TestYmlExtensionIsAlsoYaml(t *testing.T) {
	script, err := ParseScriptFromFile(writeScript(t, "dice.yml", yamlScript))
	assert.NoError(t, err)
	assert.Len(t, script.Runs, 1)
}

// One schema, one set of messages: a YAML script is rejected in the same terms
// as the JSON it converts to, using the JSON paths the schema knows about.
func TestYamlIsValidatedByTheSameSchema(t *testing.T) {
	invalid := `version: "0.3"
runs:
  - arguments: []
    exitCode: zero
    steps:
      - line: hello
`
	_, err := ParseScriptFromFile(writeScript(t, "bad.yaml", invalid))
	assert.Error(t, err)
	expected := `Script validation errors:
runs.0.exitCode: Invalid type. Expected: number, given: string`
	assert.Equal(t, expected, err.Error())
}

func TestYamlVersionIsCheckedTheSameWay(t *testing.T) {
	_, err := ParseScriptFromFile(writeScript(t, "old.yaml", "version: \"0.0\"\nruns: []\n"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), `unrecognised script format "0.0"`)
}

func TestMalformedYamlIsReported(t *testing.T) {
	_, err := ParseScriptFromFile(writeScript(t, "broken.yaml", "runs: [\n  - name: unterminated\n"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not read")
	assert.Contains(t, err.Error(), "as YAML")
}

// YAML allows a mapping key of any type and JSON does not. encoding/json only
// says it met a map[interface{}]interface{}, which is not much help.
func TestNonStringKeysAreReported(t *testing.T) {
	_, err := ParseScriptFromFile(writeScript(t, "keys.yaml", "version: \"0.3\"\n1: two\n"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "every key in a script has to be a string")
}

// An extension prescript does not recognise is read as JSON, which is what
// record writes and what every script written before this was.
func TestUnknownExtensionIsReadAsJson(t *testing.T) {
	contents := `{"version": "0.1", "runs": [{"arguments": [], "exitCode": 0, "steps": [{"line": "hello"}]}]}`
	script, err := ParseScriptFromFile(writeScript(t, "script.json", contents))
	assert.NoError(t, err)
	assert.Len(t, script.Runs, 1)
}
