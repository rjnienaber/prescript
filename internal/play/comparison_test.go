package play

import (
	"strings"
	"testing"

	"github.com/rjnienaber/prescript/internal/script"
	"github.com/rjnienaber/prescript/internal/utils"
	"github.com/stretchr/testify/assert"
)

// A transcript two runs can share, so a comparison between them is about the
// implementations rather than about the scripts.
func transcriptSteps() []script.Step {
	return []script.Step{
		{Line: "HOW MANY ROLLS? ", Input: "5000"},
		{Line: " 7             807 "},
	}
}

func comparableRun(name string) script.Run {
	return script.Run{Name: name, Steps: transcriptSteps()}
}

func matchedOutcome(name string) Outcome {
	return Outcome{Name: name, ExitCode: utils.SUCCESS}
}

func divergedOutcome(name string, divergence Divergence) Outcome {
	return Outcome{Name: name, ExitCode: utils.CLI_ERROR, Failure: &divergence}
}

func atStep(index int, received ...string) Divergence {
	divergence := Divergence{
		Mode:      NoMatch,
		Detail:    "nothing matched it within 5s",
		StepIndex: index,
		StepCount: len(transcriptSteps()),
		Expected:  transcriptSteps()[index].Line,
		Received:  received[len(received)-1],
	}
	if len(received) > 1 {
		divergence.Context = received[:len(received)-1]
	}
	return divergence
}

// The point of grouping: ports of the same program get the same thing wrong,
// and a report that says it once is a finding while a report that says it
// three times is a list.
func TestPortsThatDivergeIdenticallyAreOneFinding(t *testing.T) {
	t.Parallel()

	runs := []script.Run{comparableRun("basic"), comparableRun("ruby"), comparableRun("python")}
	outcomes := []Outcome{
		matchedOutcome("basic"),
		divergedOutcome("ruby", atStep(1, "7 807")),
		divergedOutcome("python", atStep(1, "7 807")),
	}

	comparison := Compare(runs, outcomes)

	assert.Len(t, comparison.Groups, 1)
	assert.Equal(t, []string{"ruby", "python"}, comparison.Groups[0].Ports)
	assert.Contains(t, comparison.Report(), `  ruby, python  "7 807"`)
}

func TestPortsThatDivergeDifferentlyAreSeparateFindings(t *testing.T) {
	t.Parallel()

	runs := []script.Run{comparableRun("basic"), comparableRun("ruby"), comparableRun("python")}
	outcomes := []Outcome{
		matchedOutcome("basic"),
		divergedOutcome("ruby", atStep(1, "7 807")),
		divergedOutcome("python", atStep(1, "7  807")),
	}

	comparison := Compare(runs, outcomes)

	assert.Len(t, comparison.Groups, 2)
}

// The earliest divergence is the one to read: a later one is often what the
// earlier one caused.
func TestFindingsAreOrderedByStep(t *testing.T) {
	t.Parallel()

	runs := []script.Run{comparableRun("basic"), comparableRun("ruby"), comparableRun("node")}
	outcomes := []Outcome{
		matchedOutcome("basic"),
		divergedOutcome("ruby", atStep(1, "7 807")),
		divergedOutcome("node", atStep(0, "HOW MANY ROLLS?")),
	}

	comparison := Compare(runs, outcomes)

	assert.Equal(t, []int{0, 1}, []int{comparison.Groups[0].StepIndex, comparison.Groups[1].StepIndex})
	report := comparison.Report()
	assert.Less(t, strings.Index(report, "step 1 of 2"), strings.Index(report, "step 2 of 2"))
}

// Both sides, aligned, with the reference above: what the issue asked for.
func TestReportShowsBothSidesAgainstTheReference(t *testing.T) {
	t.Parallel()

	runs := []script.Run{comparableRun("basic"), comparableRun("ruby")}
	outcomes := []Outcome{matchedOutcome("basic"), divergedOutcome("ruby", atStep(1, "7 807"))}

	report := Compare(runs, outcomes).Report()

	assert.Contains(t, report, "=== comparison against basic ===")
	assert.Contains(t, report, "step 2 of 2 (no-match, formatting)")
	assert.Contains(t, report, `  basic  " 7             807 "`)
	assert.Contains(t, report, `  ruby   "7 807"`)
	assert.Contains(t, report, "^ first difference")
}

// A port that printed several lines where one was expected: which of them was
// meant to be the step is the question, so all of them are shown and none is
// pointed at.
func TestSeveralReceivedLinesAreShownWithoutAMarker(t *testing.T) {
	t.Parallel()

	runs := []script.Run{comparableRun("basic"), comparableRun("ruby")}
	outcomes := []Outcome{
		matchedOutcome("basic"),
		divergedOutcome("ruby", atStep(1, "7 807", "and then some")),
	}

	report := Compare(runs, outcomes).Report()

	assert.Contains(t, report, `  ruby   "7 807"`)
	assert.Contains(t, report, `         "and then some"`)
	assert.NotContains(t, report, "first difference")
}

// Step 7 of one transcript has nothing to do with step 7 of another, so lining
// the two up would report a difference that is an artefact of the script.
func TestAPortPlayingADifferentTranscriptIsNotCompared(t *testing.T) {
	t.Parallel()

	other := script.Run{Name: "lua", Steps: []script.Step{{Line: "something else"}}}
	runs := []script.Run{comparableRun("basic"), other}
	outcomes := []Outcome{matchedOutcome("basic"), divergedOutcome("lua", atStep(0, "nope"))}

	comparison := Compare(runs, outcomes)

	assert.Empty(t, comparison.Groups)
	assert.Equal(t, []string{"lua"}, comparison.Incomparable)
	assert.Contains(t, comparison.Report(), "not comparable, they play a different transcript: lua")
}

// A port that never started has not disagreed with anything, and saying it did
// would be the kind of confident wrong answer this tool exists to catch.
func TestAPortThatNeverRanIsNotADivergence(t *testing.T) {
	t.Parallel()

	runs := []script.Run{comparableRun("basic"), comparableRun("go")}
	outcomes := []Outcome{matchedOutcome("basic"), {Name: "go", ExitCode: utils.USER_ERROR}}

	comparison := Compare(runs, outcomes)

	assert.Empty(t, comparison.Groups)
	assert.Equal(t, []string{"go"}, comparison.NotPlayed)
	assert.Contains(t, comparison.Report(), "could not be played: go")
}

// If the reference no longer matches its own transcript then the environment
// has moved, and every port's divergence is measured against something that is
// itself wrong.
func TestNothingIsComparedWhenTheReferenceItselfDiverged(t *testing.T) {
	t.Parallel()

	runs := []script.Run{comparableRun("basic"), comparableRun("ruby")}
	outcomes := []Outcome{
		divergedOutcome("basic", atStep(0, "HOW MANY ROLLS?")),
		divergedOutcome("ruby", atStep(1, "7 807")),
	}

	comparison := Compare(runs, outcomes)

	assert.NotNil(t, comparison.ReferenceFailure)
	assert.Empty(t, comparison.Groups)
	report := comparison.Report()
	assert.Contains(t, report, "basic did not match its own transcript")
	assert.NotContains(t, report, "ruby")
}

func TestEveryPortMatching(t *testing.T) {
	t.Parallel()

	runs := []script.Run{comparableRun("basic"), comparableRun("ruby"), comparableRun("python")}
	outcomes := []Outcome{matchedOutcome("basic"), matchedOutcome("ruby"), matchedOutcome("python")}

	report := Compare(runs, outcomes).Report()

	assert.Contains(t, report, "no port diverged from the reference")
	assert.Contains(t, report, "matched: ruby, python")
}

// A run that got through every step and then hung has no line to put opposite
// the reference, so it says what it did instead of finishing.
func TestARunThatWentWrongAfterTheLastStep(t *testing.T) {
	t.Parallel()

	hung := Divergence{
		Mode:      Hung,
		Detail:    "the executable had not exited after 5s",
		StepIndex: 2,
		StepCount: 2,
	}
	runs := []script.Run{comparableRun("basic"), comparableRun("ruby")}
	outcomes := []Outcome{matchedOutcome("basic"), divergedOutcome("ruby", hung)}

	report := Compare(runs, outcomes).Report()

	assert.Contains(t, report, "after all 2 steps (hung)")
	assert.Contains(t, report, "the executable had not exited after 5s")
}

// Columns are shared across findings so the quoted lines stay under each
// other, and a reader compares down the page instead of re-finding the column.
func TestColumnsLineUpAcrossFindings(t *testing.T) {
	t.Parallel()

	runs := []script.Run{comparableRun("basic"), comparableRun("ruby"), comparableRun("a-long-port-name")}
	outcomes := []Outcome{
		matchedOutcome("basic"),
		divergedOutcome("ruby", atStep(1, "7 807")),
		divergedOutcome("a-long-port-name", atStep(0, "HOW MANY ROLLS?")),
	}

	report := Compare(runs, outcomes).Report()

	columns := map[int]bool{}
	for _, line := range strings.Split(report, "\n") {
		if quote := strings.Index(line, `"`); strings.HasPrefix(line, "  ") && quote > 0 {
			columns[quote] = true
		}
	}
	assert.Len(t, columns, 1)
}

func drew(count int) *int { return &count }

// BASIC's PRINT puts a space before a non-negative number and another after
// it, and ports get that wrong constantly. Told apart from a wrong number, the
// whole class can be filed once per language instead of once per program.
func TestWhitespaceOnlyDivergenceIsFormatting(t *testing.T) {
	t.Parallel()

	runs := []script.Run{comparableRun("basic"), comparableRun("ruby")}
	outcomes := []Outcome{matchedOutcome("basic"), divergedOutcome("ruby", atStep(1, "7 807"))}

	comparison := Compare(runs, outcomes)

	assert.Equal(t, Formatting, comparison.Groups[0].Classification.Shape)
}

// Where a line breaks is layout by any reading, so a port that split one
// logical line across two prints is not accused of computing something else.
func TestALineBrokenInTwoIsStillFormatting(t *testing.T) {
	t.Parallel()

	runs := []script.Run{comparableRun("basic"), comparableRun("ruby")}
	outcomes := []Outcome{matchedOutcome("basic"), divergedOutcome("ruby", atStep(1, "7", "807"))}

	comparison := Compare(runs, outcomes)

	assert.Equal(t, Formatting, comparison.Groups[0].Classification.Shape)
}

// Case is text, not layout. A port that shouts where the reference whispers
// has changed what it printed.
func TestADifferentNumberOrDifferentCaseIsSemantic(t *testing.T) {
	t.Parallel()

	for _, received := range []string{" 7             806 ", " 7   eight-oh-seven "} {
		runs := []script.Run{comparableRun("basic"), comparableRun("ruby")}
		outcomes := []Outcome{matchedOutcome("basic"), divergedOutcome("ruby", atStep(1, received))}

		comparison := Compare(runs, outcomes)

		assert.Equal(t, Semantic, comparison.Groups[0].Classification.Shape, received)
	}
}

// A pattern is not a line, so there is nothing to normalise and no honest
// answer to give.
func TestARedactedStepIsNotClassifiedByShape(t *testing.T) {
	t.Parallel()

	divergence := atStep(1, "7 807")
	divergence.Redacted = true
	runs := []script.Run{comparableRun("basic"), comparableRun("ruby")}
	outcomes := []Outcome{matchedOutcome("basic"), divergedOutcome("ruby", divergence)}

	comparison := Compare(runs, outcomes)

	assert.Equal(t, UnknownShape, comparison.Groups[0].Classification.Shape)
	assert.NotContains(t, comparison.Report(), "semantic")
}

// A port that had drawn nothing cannot be diverging over the numbers it drew,
// and that is true whether or not the reference can say what it drew.
func TestAPortThatDrewNothingSaysSoWithoutTheReference(t *testing.T) {
	t.Parallel()

	ruby := divergedOutcome("ruby", atStep(1, "seven"))
	ruby.Draws = drew(0)
	runs := []script.Run{comparableRun("basic"), comparableRun("ruby")}
	outcomes := []Outcome{matchedOutcome("basic"), ruby}

	comparison := Compare(runs, outcomes)

	assert.Equal(t, NoDraws, comparison.Groups[0].Classification.Draws)
	assert.Contains(t, comparison.Report(), "(no-match, semantic, no-draws)")
}

// Same numbers asked for in the same places: the difference is in what the
// port did with them, which is a logic bug worth filing.
func TestDrawingAsMuchAsTheReferenceIsALogicBug(t *testing.T) {
	t.Parallel()

	reference := matchedOutcome("basic")
	reference.Draws = drew(12)
	ruby := divergedOutcome("ruby", atStep(1, "seven"))
	ruby.Draws = drew(12)

	comparison := Compare([]script.Run{comparableRun("basic"), comparableRun("ruby")},
		[]Outcome{reference, ruby})

	assert.Equal(t, SameDraws, comparison.Groups[0].Classification.Draws)
}

func TestDrawingADifferentAmountFromTheReferenceNeedsAHuman(t *testing.T) {
	t.Parallel()

	reference := matchedOutcome("basic")
	reference.Draws = drew(12)
	ruby := divergedOutcome("ruby", atStep(1, "seven"))
	ruby.Draws = drew(9)

	comparison := Compare([]script.Run{comparableRun("basic"), comparableRun("ruby")},
		[]Outcome{reference, ruby})

	assert.Equal(t, DifferentDraws, comparison.Groups[0].Classification.Draws)
}

// Two ports printing the same thing at the same step having drawn different
// amounts is itself the finding, and it does not need the reference to say it.
func TestPortsInOneFindingThatDrewDifferentAmounts(t *testing.T) {
	t.Parallel()

	ruby := divergedOutcome("ruby", atStep(1, "seven"))
	ruby.Draws = drew(4)
	python := divergedOutcome("python", atStep(1, "seven"))
	python.Draws = drew(9)

	comparison := Compare(
		[]script.Run{comparableRun("basic"), comparableRun("ruby"), comparableRun("python")},
		[]Outcome{matchedOutcome("basic"), ruby, python})

	assert.Len(t, comparison.Groups, 1)
	assert.Equal(t, DifferentDraws, comparison.Groups[0].Classification.Draws)
}

// The reference interpreter does not report its draws, so most real runs
// cannot answer this axis. Saying "unknown" on every line of every report
// would be noise, and a token that is always there carries nothing.
func TestAnUnknownDrawCountIsLeftOutRatherThanNamed(t *testing.T) {
	t.Parallel()

	ruby := divergedOutcome("ruby", atStep(1, "seven"))
	ruby.Draws = drew(9)

	comparison := Compare([]script.Run{comparableRun("basic"), comparableRun("ruby")},
		[]Outcome{matchedOutcome("basic"), ruby})

	assert.Equal(t, UnknownDraws, comparison.Groups[0].Classification.Draws)
	report := comparison.Report()
	assert.Contains(t, report, "(no-match, semantic)")
	assert.NotContains(t, report, "unknown")
}
