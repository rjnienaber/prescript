package script

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func parseWithRedactions(t *testing.T, redactions string, line string) (Step, error) {
	t.Helper()
	document := `{
  "version": "0.4",
  "redactions": ` + redactions + `,
  "runs": [{"arguments": [], "exitCode": 0, "steps": [{"line": ` + line + `}]}]
}`
	script, err := ParseScriptFromBytes([]byte(document))
	if err != nil {
		return Step{}, err
	}
	return script.Runs[0].Steps[0], nil
}

func TestRedactionMatchesTheVaryingPart(t *testing.T) {
	step, err := parseWithRedactions(t, `{"rolls": "[0-9]+"}`, `" 7             {{rolls}} "`)
	assert.NoError(t, err)
	assert.True(t, step.Redacted)
	assert.True(t, step.LineRegex.MatchString(" 7             807 "))
	assert.True(t, step.LineRegex.MatchString(" 7             12 "))
}

// The literal parts around a placeholder are quoted, so punctuation in the
// expected output is punctuation and not regex syntax.
func TestLiteralPartsAreNotTreatedAsPattern(t *testing.T) {
	step, err := parseWithRedactions(t, `{"n": "[0-9]+"}`, `"total (spots): {{n}}."`)
	assert.NoError(t, err)
	assert.True(t, step.LineRegex.MatchString("total (spots): 12."))
	assert.False(t, step.LineRegex.MatchString("total Xspots): 12."))
	assert.False(t, step.LineRegex.MatchString("total (spots): 12X"))
}

// A redacted line stands for the whole line, as a literal one does. Without
// anchoring it would match any line that merely contained it.
func TestRedactedLineIsAnchoredAtBothEnds(t *testing.T) {
	step, err := parseWithRedactions(t, `{"n": "[0-9]+"}`, `"seed {{n}}"`)
	assert.NoError(t, err)
	assert.True(t, step.LineRegex.MatchString("seed 4"))
	assert.False(t, step.LineRegex.MatchString("random seed 4"))
	assert.False(t, step.LineRegex.MatchString("seed 4 used"))
}

func TestSeveralRedactionsInOneLine(t *testing.T) {
	step, err := parseWithRedactions(t, `{"n": "[0-9]+", "word": "[A-Z]+"}`, `"{{word}} took {{n}}ms"`)
	assert.NoError(t, err)
	assert.True(t, step.LineRegex.MatchString("BUILD took 91ms"))
	assert.False(t, step.LineRegex.MatchString("BUILD took ms"))
}

func TestLineWithoutPlaceholdersStaysLiteral(t *testing.T) {
	step, err := parseWithRedactions(t, `{"n": "[0-9]+"}`, `"HOW MANY ROLLS? "`)
	assert.NoError(t, err)
	assert.False(t, step.Redacted)
	assert.Empty(t, step.LineRegex.String())
}

func TestUnknownRedactionIsRejected(t *testing.T) {
	_, err := parseWithRedactions(t, `{"rolls": "[0-9]+"}`, `"seed {{seed}}"`)
	assert.Error(t, err)
	expected := `Script validation errors:
runs.0.steps.0.line: unknown redaction "seed"; declared: "rolls"`
	assert.Equal(t, expected, err.Error())
}

func TestUnknownRedactionWithNoneDeclared(t *testing.T) {
	document := `{
  "version": "0.4",
  "runs": [{"arguments": [], "exitCode": 0, "steps": [{"line": "seed {{seed}}"}]}]
}`
	_, err := ParseScriptFromBytes([]byte(document))
	assert.Error(t, err)
	expected := `Script validation errors:
runs.0.steps.0.line: unknown redaction "seed"; no redactions are declared in this script`
	assert.Equal(t, expected, err.Error())
}

func TestUncompilableRedactionIsReportedOnce(t *testing.T) {
	_, err := parseWithRedactions(t, `{"n": "(unterminated"}`, `"seed {{n}} and {{n}}"`)
	assert.Error(t, err)
	// Reported against the redaction, not against every step that uses it:
	// there is one line worth fixing and it is the declaration.
	expected := `Script validation errors:
redactions.n: error parsing regexp: missing closing ): ` + "`(unterminated`"
	assert.Equal(t, expected, err.Error())
}

// Redactions are declared for the script, so runs being compared normalise the
// same things in the same way rather than each carrying its own copy.
func TestRedactionsApplyToEveryRun(t *testing.T) {
	document := `{
  "version": "0.4",
  "redactions": {"n": "[0-9]+"},
  "runs": [
    {"name": "reference", "arguments": [], "exitCode": 0, "steps": [{"line": "rolled {{n}}"}]},
    {"name": "port", "arguments": [], "exitCode": 0, "steps": [{"line": "rolled {{n}}"}]}
  ]
}`
	script, err := ParseScriptFromBytes([]byte(document))
	assert.NoError(t, err)
	for _, run := range script.Runs {
		assert.True(t, run.Steps[0].Redacted, run.Name)
		assert.True(t, run.Steps[0].LineRegex.MatchString("rolled 6"), run.Name)
	}
}
