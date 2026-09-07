package play

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rjnienaber/prescript/internal/script"
	"github.com/rjnienaber/prescript/internal/utils"
)

// Comparison is what a script with more than one run was written to produce:
// not a verdict on each implementation separately, but the places where they
// stopped agreeing.
//
// The first run is the reference. That is not a rule imposed on the format so
// much as a description of how such a script comes about -- it is recorded
// from the implementation whose behaviour is being treated as correct, and the
// ports are added underneath. It also means the reference is played like any
// other run, which matters: if it no longer matches its own transcript then
// the environment has moved, and every port's "divergence" is measured against
// something that is itself wrong.
type Comparison struct {
	Reference string

	// ReferenceFailure is nil when the reference matched its transcript. When
	// it is set, nothing else here was compared.
	ReferenceFailure *Divergence

	Groups []DivergenceGroup

	// Matched, Incomparable and NotPlayed are port names. A port is
	// incomparable when its steps are not the reference's steps: step 7 of one
	// transcript has nothing to do with step 7 of another, and lining the two
	// up would produce a difference that is an artefact of the script.
	Matched      []string
	Incomparable []string
	NotPlayed    []string
}

// DivergenceGroup is one finding: the ports that left the reference at the
// same step, in the same way, printing the same thing.
//
// Ports are grouped rather than listed one by one because they are not
// independent observations. A dozen ports of the same program getting BASIC's
// leading space before a number wrong is one bug repeated twelve times, and a
// report that says it twelve times is a report nobody reads to the end.
type DivergenceGroup struct {
	StepIndex int
	StepCount int
	Mode      FailureMode
	Expected  string
	Redacted  bool

	// Received is everything the ports printed since they last agreed with the
	// transcript. When every step matched and the run went wrong afterwards
	// there is nothing left to have printed, and this holds the detail -- what
	// the program did instead of finishing -- so the group still says
	// something.
	Received []string
	Matched  bool

	Ports []string
}

// Compare lines up what each run did against the reference's transcript. It
// reads outcomes rather than replaying anything: the runs have already been
// played, in the order the script names them, and outcomes[i] belongs to
// runs[i].
func Compare(runs []script.Run, outcomes []Outcome) Comparison {
	comparison := Comparison{Reference: outcomes[0].Name, ReferenceFailure: outcomes[0].Failure}
	if comparison.ReferenceFailure != nil {
		return comparison
	}

	reference := transcript(runs[0])
	// Keyed by what makes two divergences the same finding.
	index := map[string]int{}

	for position, outcome := range outcomes[1:] {
		run := runs[position+1]

		switch {
		case transcript(run) != reference:
			comparison.Incomparable = append(comparison.Incomparable, outcome.Name)
		case outcome.Failure == nil && outcome.ExitCode == utils.SUCCESS:
			comparison.Matched = append(comparison.Matched, outcome.Name)
		case outcome.Failure == nil:
			comparison.NotPlayed = append(comparison.NotPlayed, outcome.Name)
		default:
			comparison.add(index, outcome.Name, *outcome.Failure)
		}
	}

	// Earliest step first, and within a step the order the ports appear in the
	// script. A later divergence is often a consequence of an earlier one --
	// the port that got the prompt wrong is the port that then answered the
	// wrong question -- so the first one is the one to read, and it should not
	// be somewhere in the middle because of which port happened to hit it.
	sort.SliceStable(comparison.Groups, func(i, j int) bool {
		return comparison.Groups[i].StepIndex < comparison.Groups[j].StepIndex
	})

	return comparison
}

func (comparison *Comparison) add(index map[string]int, port string, divergence Divergence) {
	group := DivergenceGroup{
		StepIndex: divergence.StepIndex,
		StepCount: divergence.StepCount,
		Mode:      divergence.Mode,
		Expected:  divergence.Expected,
		Redacted:  divergence.Redacted,
		Received:  divergence.Lines(),
		Matched:   divergence.Matched(),
	}
	if group.Matched {
		group.Received = []string{divergence.Detail}
	}

	key := fmt.Sprintf("%d\x00%s\x00%s", group.StepIndex, group.Mode,
		strings.Join(group.Received, "\x00"))
	if at, seen := index[key]; seen {
		comparison.Groups[at].Ports = append(comparison.Groups[at].Ports, port)
		return
	}

	group.Ports = []string{port}
	index[key] = len(comparison.Groups)
	comparison.Groups = append(comparison.Groups, group)
}

// transcript is what makes two runs comparable: the same expectations, fed the
// same input. Everything else about a run -- which executable, which
// environment, which terminal -- is exactly what the comparison is about, and
// two runs identical in those respects would have nothing to say.
func transcript(run script.Run) string {
	var signature strings.Builder
	for _, step := range run.Steps {
		signature.WriteString(step.Line)
		signature.WriteString("\x00")
		signature.WriteString(step.Input)
		signature.WriteString("\x00\x00")
	}
	return signature.String()
}

// Report writes the comparison out for a person to read.
func (comparison Comparison) Report() string {
	var report strings.Builder
	fmt.Fprintf(&report, "\n=== comparison against %s ===\n\n", comparison.Reference)

	if comparison.ReferenceFailure != nil {
		fmt.Fprintf(&report, "%s did not match its own transcript, so there was nothing for the\n"+
			"ports to be compared against. Its own failure is above.\n", comparison.Reference)
		return report.String()
	}

	// One column width for the whole report rather than one per group, so the
	// quoted lines stay under each other and a reader compares down the page
	// instead of re-finding the column at every step.
	width := comparison.labelWidth()
	for _, group := range comparison.Groups {
		report.WriteString(group.report(comparison.Reference, width))
		report.WriteString("\n")
	}

	// Said only of the ports that were actually compared: one that plays a
	// different transcript, or that never started, has not agreed with the
	// reference -- it has not been asked. Whichever line is used has to be
	// true on its own, because it is the line that gets quoted.
	if len(comparison.Groups) == 0 && len(comparison.Matched) > 0 {
		if len(comparison.Incomparable) > 0 || len(comparison.NotPlayed) > 0 {
			report.WriteString("none of the ports that were compared diverged from the reference\n")
		} else {
			report.WriteString("no port diverged from the reference\n")
		}
	}

	writeNames(&report, "matched", comparison.Matched)
	writeNames(&report, "not comparable, they play a different transcript", comparison.Incomparable)
	writeNames(&report, "could not be played", comparison.NotPlayed)

	return report.String()
}

func (comparison Comparison) labelWidth() int {
	width := len(comparison.Reference)
	for _, group := range comparison.Groups {
		if label := len(strings.Join(group.Ports, ", ")); label > width {
			width = label
		}
	}
	return width
}

func writeNames(report *strings.Builder, label string, names []string) {
	if len(names) == 0 {
		return
	}
	fmt.Fprintf(report, "%s: %s\n", label, strings.Join(names, ", "))
}

// report lays a group out with the reference's line above the ports' output,
// so the difference is read down the page rather than across it. Both are
// quoted for the same reason a single failure report quotes them: a trailing
// space is the difference that matters most and shows least.
func (group DivergenceGroup) report(reference string, width int) string {
	ports := strings.Join(group.Ports, ", ")
	indent := len("  ") + width + len("  ")

	var report strings.Builder
	if group.Matched {
		fmt.Fprintf(&report, "after all %d steps (%s)\n", group.StepCount, group.Mode)
		fmt.Fprintf(&report, "  %-*s  %s\n", width, ports, strings.Join(group.Received, " "))
		return report.String()
	}

	fmt.Fprintf(&report, "step %d of %d (%s)\n", group.StepIndex+1, group.StepCount, group.Mode)
	if group.Redacted {
		fmt.Fprintf(&report, "  %-*s  %q  (with redactions applied)\n", width, reference, group.Expected)
	} else {
		fmt.Fprintf(&report, "  %-*s  %q\n", width, reference, group.Expected)
	}

	for index, line := range group.Received {
		label := ""
		if index == 0 {
			label = ports
		}
		fmt.Fprintf(&report, "  %-*s  %q\n", width, label, line)
	}

	// A caret under one of several lines would be asserting which of them was
	// meant to be the step, and that is the thing in question. With one line
	// there is nothing to assert, and pointing at the byte where it went wrong
	// is the whole reason a report quotes it.
	if !group.Redacted && len(group.Received) == 1 {
		report.WriteString(differenceMarker(group.Expected, group.Received[0], indent))
	}

	return report.String()
}
