package script

import (
	"testing"
	"time"

	cfg "github.com/rjnienaber/prescript/internal/config"
	"github.com/rjnienaber/prescript/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestBuildBasicScript(t *testing.T) {
	config := cfg.RecordConfig{ExecutablePath: "input.sh", Arguments: []string{}}
	lines := []utils.CapturedLine{
		{Value: "Hello!", LineType: utils.Output},
		{Value: "Please enter your name: ", LineType: utils.Output},
		{Value: "Richard", LineType: utils.Input},
		{Value: "How do you do, Richard", LineType: utils.Output},
	}
	now := time.Date(2009, 11, 17, 20, 34, 58, 0, time.UTC)

	script, err := BuildScriptJson(config, lines, 0, now, &utils.CustomLogger{})

	assert.NoError(t, err)

	expected := `{
  "version": "` + CurrentVersion + `",
  "runs": [
    {
      "timestamp": "2009-11-17T20:34:58Z",
      "executable": "input.sh",
      "arguments": [],
      "exitCode": 0,
      "steps": [
        {
          "line": "Please enter your name: ",
          "input": "Richard"
        }
      ]
    }
  ]
}`
	assert.Equal(t, expected, script)
}

// A recording made through pipes is only reproducible if the script says so:
// replayed under the default the program would find a terminal it was never
// recorded against, and may not print the same thing.
func TestBuildRecordsPipesButNotThePtyDefault(t *testing.T) {
	lines := []utils.CapturedLine{
		{Value: "Name: ", LineType: utils.Output},
		{Value: "Richard", LineType: utils.Input},
	}
	now := time.Date(2009, 11, 17, 20, 34, 58, 0, time.UTC)

	for terminal, expected := range map[string]string{"": "", "pty": "", "pipes": `"terminal": "pipes"`} {
		config := cfg.RecordConfig{ExecutablePath: "input.sh", Arguments: []string{}, Terminal: terminal}

		script, err := BuildScriptJson(config, lines, 0, now, &utils.CustomLogger{})

		assert.NoError(t, err)
		if expected == "" {
			assert.NotContains(t, script, "terminal", "terminal %q", terminal)
		} else {
			assert.Contains(t, script, expected, "terminal %q", terminal)
		}
	}
}
