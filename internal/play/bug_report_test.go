package play

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	cfg "github.com/rjnienaber/prescript/internal/config"
	"github.com/rjnienaber/prescript/internal/script"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func reportConfig() cfg.PlayConfig {
	return cfg.PlayConfig{ScriptFile: "dice.yaml", Terminal: "pipes"}
}

// A report about the second thing that went wrong is a report about a
// consequence. Everything after the first divergence was produced by a program
// that had already stopped running the same script.
func TestOnlyTheFirstDivergenceIsReported(t *testing.T) {
	t.Parallel()

	runs := []script.Run{comparableRun("basic"), comparableRun("ruby"), comparableRun("python")}
	outcomes := []Outcome{
		matchedOutcome("basic"),
		divergedOutcome("python", atStep(1, "7 807")),
		divergedOutcome("ruby", atStep(0, "HOW MANY ROLLS?")),
	}

	report, found := NewBugReport(reportConfig(), runs, outcomes, nil)
	require.True(t, found)

	assert.Equal(t, []string{"ruby"}, report.Finding.Ports)
	assert.Contains(t, report.Markdown(), "# ruby: no-match at step 1 of 2")
	assert.NotContains(t, report.Markdown(), "python")
	assert.NotContains(t, report.Markdown(), "step 2 of 2")
}

func TestBothSidesAndTheClassificationAreInTheReport(t *testing.T) {
	t.Parallel()

	runs := []script.Run{comparableRun("basic"), comparableRun("ruby")}
	outcomes := []Outcome{
		matchedOutcome("basic"),
		divergedOutcome("ruby", atStep(1, "7 807")),
	}

	markdown := markdownFor(t, reportConfig(), runs, outcomes)

	assert.Contains(t, markdown, "Classified `formatting`")
	assert.Contains(t, markdown, "  basic  \" 7             807 \"")
	assert.Contains(t, markdown, "  ruby   \"7 807\"")
}

// With one run there is no implementation being treated as correct, and
// naming one that was never played would be a lie about where the expected
// side came from.
func TestASingleRunIsReportedAgainstItsTranscript(t *testing.T) {
	t.Parallel()

	runs := []script.Run{comparableRun("ruby")}
	outcomes := []Outcome{divergedOutcome("ruby", atStep(1, "7 807"))}

	report, found := NewBugReport(reportConfig(), runs, outcomes, nil)
	require.True(t, found)

	assert.Equal(t, "the transcript", report.Reference)
	assert.Contains(t, report.Markdown(), "against `the transcript`")
}

// A reference that no longer matches its own recording is the more urgent
// finding, not a reason to write nothing: the environment has moved under a
// script that used to pass.
func TestAReferenceThatDivergedIsItselfTheReport(t *testing.T) {
	t.Parallel()

	runs := []script.Run{comparableRun("basic"), comparableRun("ruby")}
	outcomes := []Outcome{
		divergedOutcome("basic", atStep(1, "7 806")),
		matchedOutcome("ruby"),
	}

	report, found := NewBugReport(reportConfig(), runs, outcomes, nil)
	require.True(t, found)

	assert.Equal(t, []string{"basic"}, report.Finding.Ports)
	assert.Equal(t, "the transcript", report.Reference)
}

func TestThereIsNothingToReportWhenEveryRunMatched(t *testing.T) {
	t.Parallel()

	runs := []script.Run{comparableRun("basic"), comparableRun("ruby")}
	outcomes := []Outcome{matchedOutcome("basic"), matchedOutcome("ruby")}

	_, found := NewBugReport(reportConfig(), runs, outcomes, nil)
	assert.False(t, found)
}

// The repro line is the command that was actually run, minus the two things
// that are about this machine rather than about the bug: how the binary was
// reached, and where the checkout happens to live.
func TestTheReproCommandIsPasteable(t *testing.T) {
	t.Parallel()

	working, err := os.Getwd()
	require.NoError(t, err)

	runs := []script.Run{comparableRun("ruby")}
	outcomes := []Outcome{divergedOutcome("ruby", atStep(1, "7 807"))}
	commandLine := []string{"/Users/someone/go/bin/prescript", "play",
		filepath.Join(working, "dice.yaml"), "--tape", "/elsewhere/dice.tape"}

	report, found := NewBugReport(reportConfig(), runs, outcomes, commandLine)
	require.True(t, found)

	assert.Equal(t,
		[]string{"prescript", "play", "dice.yaml", "--tape", "/elsewhere/dice.tape"},
		report.Command)
	assert.Contains(t, report.Markdown(), "prescript play dice.yaml --tape /elsewhere/dice.tape")
}

func TestAnUnpinnedRunSaysSoRatherThanLeavingAGap(t *testing.T) {
	t.Setenv("PRESCRIPT_IMAGE", "")

	markdown := markdownFor(t, reportConfig(), []script.Run{comparableRun("ruby")},
		[]Outcome{divergedOutcome("ruby", atStep(1, "7 807"))})

	assert.Contains(t, markdown, "| image | `none; this did not run in the pinned environment` |")
	assert.NotContains(t, markdown, "make pull_image")
}

// A digest is the only thing that makes the rest of the report reproducible,
// so when there is one it becomes the first line of the repro block.
func TestAPinnedRunPullsTheImageFirst(t *testing.T) {
	t.Setenv("PRESCRIPT_IMAGE", "ghcr.io/rjnienaber/prescript-env@sha256:abc123")

	markdown := markdownFor(t, reportConfig(), []script.Run{comparableRun("ruby")},
		[]Outcome{divergedOutcome("ruby", atStep(1, "7 807"))})

	assert.Contains(t, markdown,
		"make pull_image PUBLISHED_REF=ghcr.io/rjnienaber/prescript-env@sha256:abc123")
}

// A locally built image has a config digest and no name anybody else can pull
// it by. It says which bytes ran, which is worth recording, and it is not a
// command anyone can paste.
func TestALocallyBuiltImageIsNamedButNotPulled(t *testing.T) {
	t.Setenv("PRESCRIPT_IMAGE", "sha256:9f8e7d")

	markdown := markdownFor(t, reportConfig(), []script.Run{comparableRun("ruby")},
		[]Outcome{divergedOutcome("ruby", atStep(1, "7 807"))})

	assert.Contains(t, markdown, "| image | `sha256:9f8e7d` |")
	assert.NotContains(t, markdown, "make pull_image")
}

// Without the tape the finding is not reproducible even in principle, so what
// the tape says about where it came from is quoted rather than summarised.
func TestTheTapeAndItsProvenanceAreQuoted(t *testing.T) {
	t.Parallel()

	tape := filepath.Join(t.TempDir(), "dice.tape")
	require.NoError(t, os.WriteFile(tape,
		[]byte("# recorded from vintbas 0.4.0, seed 0\n# 3 values\n0.5\n0.25\n0.75\n"), 0o600))

	config := reportConfig()
	config.Tape = tape

	markdown := markdownFor(t, config, []script.Run{comparableRun("ruby")},
		[]Outcome{divergedOutcome("ruby", atStep(1, "7 807"))})

	assert.Contains(t, markdown, "| tape | `"+tape+"` |")
	assert.Contains(t, markdown, "# recorded from vintbas 0.4.0, seed 0")
	assert.Contains(t, markdown, "# 3 values")
	assert.NotContains(t, markdown, "0.25")
}

// Without a tape the seed is what makes the run repeatable, and it belongs to
// the run rather than to prescript.
func TestTheSeedStandsInForTheTape(t *testing.T) {
	t.Parallel()

	run := comparableRun("ruby")
	run.Env = map[string]string{"PRESCRIPT_SEED": "42", "TZ": "UTC"}

	markdown := markdownFor(t, reportConfig(), []script.Run{run},
		[]Outcome{divergedOutcome("ruby", atStep(1, "7 807"))})

	assert.Contains(t, markdown, "| seed | `42` |")
	assert.Contains(t, markdown, "| TZ | `UTC` |")
}

func TestTheDrawCountIsReportedOnlyWhenItIsKnown(t *testing.T) {
	t.Parallel()

	runs := []script.Run{comparableRun("ruby")}
	outcome := divergedOutcome("ruby", atStep(1, "7 807"))

	assert.NotContains(t, markdownFor(t, reportConfig(), runs, []Outcome{outcome}), "| draws |")

	outcome.Draws = drew(12)
	assert.Contains(t, markdownFor(t, reportConfig(), runs, []Outcome{outcome}), "| draws | `12` |")
}

func TestTheTerminalAndPrescriptsOwnVersionAreReported(t *testing.T) {
	t.Parallel()

	markdown := markdownFor(t, reportConfig(), []script.Run{comparableRun("ruby")},
		[]Outcome{divergedOutcome("ruby", atStep(1, "7 807"))})

	assert.Contains(t, markdown, "| terminal | `pipes` |")
	assert.Contains(t, markdown, "| prescript |")
	assert.Contains(t, markdown, "| timeout |")
}

// A program named by a path is the thing under test. Running somebody else's
// program with an argument it never expected is not a thing to do behind their
// back, and the answer would be the port's version rather than the toolchain's
// anyway.
func TestOnlyAnExecutableOnThePathIsAskedForItsVersion(t *testing.T) {
	t.Parallel()

	assert.Empty(t, toolchainVersion("/usr/bin/ruby"))
	assert.Empty(t, toolchainVersion("./dice"))
	assert.Empty(t, toolchainVersion(""))
	assert.Empty(t, toolchainVersion("prescript-no-such-executable"))
}

func TestTheReportIsWrittenToTheNamedFile(t *testing.T) {
	t.Parallel()

	config := reportConfig()
	config.BugReport = filepath.Join(t.TempDir(), "report.md")

	written, err := WriteBugReport(config, []script.Run{comparableRun("ruby")},
		[]Outcome{divergedOutcome("ruby", atStep(1, "7 807"))}, nil)
	require.NoError(t, err)
	assert.True(t, written)

	contents, err := os.ReadFile(config.BugReport)
	require.NoError(t, err)
	assert.Contains(t, string(contents), "# ruby: no-match at step 2 of 2")
}

func markdownFor(t *testing.T, config cfg.PlayConfig, runs []script.Run, outcomes []Outcome) string {
	t.Helper()

	report, found := NewBugReport(config, runs, outcomes, nil)
	require.True(t, found)

	markdown := report.Markdown()
	// Every section is required: a report missing one of them is the kind of
	// report that gets answered with "works for me".
	for _, heading := range []string{"## The divergence", "## Reproducing it", "## The environment"} {
		assert.True(t, strings.Contains(markdown, heading), heading)
	}
	return markdown
}
