package play

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	cfg "github.com/rjnienaber/prescript/internal/config"
	"github.com/rjnienaber/prescript/internal/script"
	"github.com/rjnienaber/prescript/internal/utils"
)

// BugReport is one divergence written up so that somebody else can act on it
// without re-deriving the setup.
//
// The default answer to any cross-implementation bug report is "works for me",
// and it is usually a fair answer: the two people ran different interpreter
// builds, in different locales, against different random numbers. Only a
// complete repro block prevents it, and assembling one by hand for every
// finding is work nobody does twice. So it is assembled here, from what the
// run already knows about itself.
//
// The first divergence only. A run that went wrong at step 5 was not really
// running the program from step 6 onwards, and everything after the first
// difference is a consequence of it.
type BugReport struct {
	Script    string
	Reference string
	Finding   DivergenceGroup

	// Command reproduces the run. It is the command line prescript was given,
	// which is the honest answer to "what did you run" -- rewritten only to
	// drop the path prescript itself was invoked by and to make repository
	// paths relative, so a report can be pasted somewhere public without
	// carrying a home directory into it.
	Command []string

	// Toolchain is what the pinned environment says about itself: the base
	// image digest and the version of every interpreter in it, written at
	// build time from the binaries actually installed. Empty outside it.
	Toolchain string

	// TapeHeader is the provenance the tape carries: which interpreter
	// recorded it, against which generator, from which seed.
	TapeHeader []string

	Facts []Fact
}

// Fact is one row of the environment table. Ordered, because the order is the
// argument: what was run, then what it was run against, then what was around
// it.
type Fact struct {
	Name  string
	Value string
}

// NewBugReport picks the divergence worth reporting and gathers everything
// around it. It reports false when there is nothing to report -- every run
// passed, or the runs that failed never got as far as diverging.
func NewBugReport(config cfg.PlayConfig, runs []script.Run, outcomes []Outcome, commandLine []string) (BugReport, bool) {
	reference, finding, found := firstFinding(runs, outcomes)
	if !found {
		return BugReport{}, false
	}

	run, outcome := runFor(runs, outcomes, finding.Ports[0])
	report := BugReport{
		Script:     config.ScriptFile,
		Reference:  reference,
		Finding:    finding,
		Command:    reproCommand(commandLine),
		Toolchain:  readFile(os.Getenv("PRESCRIPT_ENVIRONMENT")),
		TapeHeader: tapeHeader(config.Tape),
	}
	report.Facts = gatherFacts(config, run, outcome)
	return report, true
}

// firstFinding is the divergence to report: the earliest one in a comparison,
// or the single run's own when there is nothing to compare it against.
func firstFinding(runs []script.Run, outcomes []Outcome) (string, DivergenceGroup, bool) {
	if len(outcomes) > 1 {
		comparison := Compare(runs, outcomes)
		if len(comparison.Groups) > 0 {
			return comparison.Reference, comparison.Groups[0], true
		}

		// A reference that does not match its own transcript is still a
		// finding, and often the more urgent one: the environment has moved
		// under a script that used to pass.
		if comparison.ReferenceFailure != nil {
			return transcriptLabel, asGroup(comparison.Reference, *comparison.ReferenceFailure), true
		}
		return "", DivergenceGroup{}, false
	}

	if len(outcomes) == 1 && outcomes[0].Failure != nil {
		return transcriptLabel, asGroup(outcomes[0].Name, *outcomes[0].Failure), true
	}
	return "", DivergenceGroup{}, false
}

// asGroup makes one run's divergence look like a finding of one, so a report
// about a single run and a report about a comparison are the same report.
// There is no reference implementation on this path, so the expected line
// speaks for itself and is labelled as what the script asked for.
func asGroup(name string, divergence Divergence) DivergenceGroup {
	group := DivergenceGroup{
		StepIndex: divergence.StepIndex,
		StepCount: divergence.StepCount,
		Mode:      divergence.Mode,
		Expected:  divergence.Expected,
		Redacted:  divergence.Redacted,
		Received:  divergence.Lines(),
		Matched:   divergence.Matched(),
		Ports:     []string{name},
	}
	if group.Matched {
		group.Received = []string{divergence.Detail}
	}
	group.Classification = group.classify(nil, []*int{nil})
	return group
}

func runFor(runs []script.Run, outcomes []Outcome, name string) (script.Run, Outcome) {
	for index, outcome := range outcomes {
		if outcome.Name == name {
			return runs[index], outcome
		}
	}
	return script.Run{}, Outcome{}
}

func gatherFacts(config cfg.PlayConfig, run script.Run, outcome Outcome) []Fact {
	facts := []Fact{
		{"prescript", utils.Version()},
		{"script", config.ScriptFile},
		{"command", strings.TrimSpace(run.Executable + " " + strings.Join(run.Arguments, " "))},
	}

	if toolchain := toolchainVersion(run.Executable); toolchain != "" {
		facts = append(facts, Fact{run.Executable, toolchain})
	}

	// An unpinned run is worth saying out loud rather than leaving as a gap.
	// It is the single likeliest reason a reader cannot reproduce what they
	// are being shown.
	if image := os.Getenv("PRESCRIPT_IMAGE"); image != "" {
		facts = append(facts, Fact{"image", image})
	} else {
		facts = append(facts, Fact{"image", "none; this did not run in the pinned environment"})
	}

	environment := childEnv(run)
	if config.Tape != "" {
		facts = append(facts, Fact{"tape", config.Tape})
	} else if seed := lookup(environment, "PRESCRIPT_SEED"); seed != "" {
		facts = append(facts, Fact{"seed", seed})
	}

	if outcome.Draws != nil {
		facts = append(facts, Fact{"draws", strconv.Itoa(*outcome.Draws)})
	}

	terminal, err := getTerminal(config, run)
	if err == nil {
		facts = append(facts, Fact{"terminal", describeTerminal(terminal)})
	}

	facts = append(facts, Fact{"timeout", config.Timeout.String()})

	for _, name := range []string{"LC_ALL", "LANG", "TZ"} {
		if value := lookup(environment, name); value != "" {
			facts = append(facts, Fact{name, value})
		}
	}

	return facts
}

func describeTerminal(terminal utils.Terminal) string {
	if terminal == utils.TerminalPipes {
		return "pipes"
	}

	columns, rows := utils.TerminalSize()
	return fmt.Sprintf("pty, %dx%d", columns, rows)
}

// childEnv is the environment the program was actually started with, which is
// prescript's own when the run declared none.
func childEnv(run script.Run) []string {
	if environment := run.Environment(os.Environ()); environment != nil {
		return environment
	}
	return os.Environ()
}

func lookup(environment []string, name string) string {
	value := ""
	// Last assignment wins, the same way exec resolves it.
	for _, entry := range environment {
		if strings.HasPrefix(entry, name+"=") {
			value = strings.TrimPrefix(entry, name+"=")
		}
	}
	return value
}

// toolchainVersion asks the interpreter what it is.
//
// Only an executable named by name is asked. One named by a path is the
// program under test rather than the toolchain under it, and running somebody
// else's program with an argument it never expected to see is not a thing to
// do behind their back.
func toolchainVersion(executable string) string {
	if executable == "" || strings.ContainsRune(executable, filepath.Separator) {
		return ""
	}

	arguments := []string{"--version"}
	if executable == "go" {
		arguments = []string{"version"}
	}

	// Bounded, because this runs after a failure and must not become a second
	// way for the same run to hang.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, err := exec.CommandContext(ctx, executable, arguments...).CombinedOutput()
	if err != nil && len(output) == 0 {
		return ""
	}

	for _, line := range strings.Split(string(output), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
	}
	return ""
}

// tapeHeader is the provenance the tape carries in its own comments. Quoted
// rather than summarised: which interpreter recorded it, built against which
// generator, from which seed is the difference between a report that can be
// reproduced and one that cannot even in principle.
func tapeHeader(path string) []string {
	contents := readFile(path)
	if contents == "" {
		return nil
	}

	var header []string
	for _, line := range strings.Split(contents, "\n") {
		if !strings.HasPrefix(line, "#") {
			break
		}
		header = append(header, line)
	}
	return header
}

func readFile(path string) string {
	if path == "" {
		return ""
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimRight(string(contents), "\n")
}

// reproCommand rewrites the command line into one a reader can paste.
//
// argv[0] becomes "prescript", because how this machine happens to reach the
// binary is not part of the bug, and every path inside the working directory
// becomes relative to it -- both so the command works from a checkout and so a
// report can be posted in public without a home directory in it.
func reproCommand(commandLine []string) []string {
	if len(commandLine) == 0 {
		return nil
	}

	working, err := os.Getwd()
	if err != nil {
		working = ""
	}

	command := []string{"prescript"}
	for _, argument := range commandLine[1:] {
		command = append(command, relativise(argument, working))
	}
	return command
}

func relativise(argument string, working string) string {
	if working == "" || !filepath.IsAbs(argument) {
		return argument
	}

	relative, err := filepath.Rel(working, argument)
	if err != nil || strings.HasPrefix(relative, "..") {
		return argument
	}
	return relative
}

// transcriptLabel is what the expected side is called when there is no
// reference implementation to name -- a single run, or a comparison whose
// reference did not match its own recording. The script is then the only
// authority in the room, and saying so is more honest than naming a port that
// was not consulted.
const transcriptLabel = "the transcript"

// Markdown is the report as a file. Markdown because it is what the issue
// trackers these end up in accept, and it stays readable in a terminal for
// anyone who just cats it.
//
// The order is the order it gets read in: what went wrong, how to see it
// again, and only then the environment it went wrong in. A reader who already
// knows the bug stops after the first section; a reader who says "works for
// me" is looking for the third.
func (report BugReport) Markdown() string {
	var out strings.Builder

	fmt.Fprintf(&out, "# %s\n\n", report.title())
	fmt.Fprintf(&out, "In `%s`, against `%s`.\n", report.Script, report.Reference)
	if tokens := report.Finding.Classification.Tokens(); len(tokens) > 0 {
		fmt.Fprintf(&out, "Classified `%s`.\n", strings.Join(tokens, ", "))
	}

	out.WriteString("\n## The divergence\n\n")
	out.WriteString("```\n")
	out.WriteString(report.Finding.report(report.Reference, len(report.Reference)))
	out.WriteString("```\n")

	// Only the first. A run that went wrong at step 5 was not really running
	// the same program from step 6 onwards, and a report that lists every
	// later difference is asking its reader to work out which one is the
	// cause.
	if later := report.Finding.StepCount - report.Finding.StepIndex - 1; later > 0 && !report.Finding.Matched {
		fmt.Fprintf(&out, "\nThis is the first divergence; %s never reached.\n", pluralSteps(later))
	}

	out.WriteString("\n## Reproducing it\n\n")
	out.WriteString("```sh\n")
	for _, line := range report.repro() {
		out.WriteString(line + "\n")
	}
	out.WriteString("```\n")

	out.WriteString("\n## The environment\n\n")
	out.WriteString(factTable(report.Facts))

	if len(report.TapeHeader) > 0 {
		out.WriteString("\nThe tape says of itself:\n\n```\n")
		for _, line := range report.TapeHeader {
			out.WriteString(line + "\n")
		}
		out.WriteString("```\n")
	}

	if report.Toolchain != "" {
		out.WriteString("\nThe image was built with:\n\n```\n")
		out.WriteString(report.Toolchain + "\n")
		out.WriteString("```\n")
	}

	return out.String()
}

func (report BugReport) title() string {
	who := strings.Join(report.Finding.Ports, ", ")
	if report.Finding.Matched {
		return fmt.Sprintf("%s: %s after all %d steps", who, report.Finding.Mode, report.Finding.StepCount)
	}
	return fmt.Sprintf("%s: %s at step %d of %d", who, report.Finding.Mode,
		report.Finding.StepIndex+1, report.Finding.StepCount)
}

// repro is the block someone else pastes. Pulling the image comes first
// because it is the step that makes the rest mean anything.
//
// Only a registry digest earns that line. A locally built image is identified
// by its config digest, a bare sha256: with no name in front of it, which
// says which bytes ran but is not something anybody else can pull -- so it is
// named in the table and left out of the commands. A command that names an
// image nobody can fetch is worse than no command.
func (report BugReport) repro() []string {
	var lines []string
	for _, fact := range report.Facts {
		if fact.Name == "image" && strings.Contains(fact.Value, "@sha256:") {
			lines = append(lines, "make pull_image PUBLISHED_REF="+fact.Value)
		}
	}
	return append(lines, strings.Join(report.Command, " "))
}

// factTable is a two-column table without a header row, because both columns
// are labelled by what is in them and a "name | value" header adds a line
// nobody reads. GitHub needs the separator regardless.
func factTable(facts []Fact) string {
	var table strings.Builder
	table.WriteString("| | |\n| --- | --- |\n")
	for _, fact := range facts {
		fmt.Fprintf(&table, "| %s | %s |\n", fact.Name, inlineCode(fact.Value))
	}
	return table.String()
}

// inlineCode quotes a value as code so that a path, a digest or a version
// survives being rendered. A value with a backtick in it is left alone rather
// than escaped: nothing here has one, and a broken table is easier to notice
// than a silently mangled value.
func inlineCode(value string) string {
	if value == "" || strings.ContainsRune(value, '`') {
		return value
	}
	return "`" + value + "`"
}

// WriteBugReport writes the report for these outcomes, if there is one to
// write. It reports whether it wrote anything, and any error is the caller's
// to report: a bug report that could not be written is worth saying out loud,
// because the run it described has already finished and will not be repeated
// for free.
func WriteBugReport(config cfg.PlayConfig, runs []script.Run, outcomes []Outcome, commandLine []string) (bool, error) {
	report, found := NewBugReport(config, runs, outcomes, commandLine)
	if !found {
		return false, nil
	}

	if err := os.WriteFile(config.BugReport, []byte(report.Markdown()), 0o644); err != nil {
		return false, err
	}
	return true, nil
}
