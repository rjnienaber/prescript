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
  "version": "0.1",
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
