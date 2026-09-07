package play

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Divergence is the point at which a run stopped agreeing with the transcript
// it was played against: which step, what that step wanted, and what arrived
// instead.
//
// It exists as a value rather than as the string the failure report used to
// build directly, because one run's failure is a report and several runs'
// failures are a comparison. Aligning three ports at the step where each of
// them left the reference is only possible if that step is a number something
// else can group by, and the same is true of counting a corpus by what went
// wrong. The report is then a rendering of this, so the two cannot drift.
type Divergence struct {
	Mode   FailureMode
	Detail string

	// StepIndex is zero-based, and equals StepCount when every step matched
	// and the run went wrong afterwards -- it hung, or exited with the wrong
	// code. Matched says which of the two this is.
	StepIndex int
	StepCount int

	// Expected and Received are empty when every step matched: there is no
	// step left to have wanted anything.
	Expected string
	Received string

	// Redacted repeats the step's own flag, because it decides both how the
	// expected line is labelled and whether pointing at a first differing
	// byte would mean anything.
	Redacted bool

	// Context is the output since the last matched step, excluding the line
	// already shown as Received.
	Context []string
}

// Matched is true when the run got through every step, and what went wrong
// happened after the last one.
func (divergence Divergence) Matched() bool {
	return divergence.StepIndex >= divergence.StepCount
}

// Lines is everything the program printed since it last agreed with the
// transcript, in the order it printed it, ending with the line the report
// names as received.
//
// A single failure report picks one of these to put opposite the expected
// line, and the rest is context. A comparison cannot: which line of several
// was meant to be the step's is exactly what is in question -- the program
// that printed its own version of the line and carried on, and the program
// that printed three lines of error and died, both arrive here -- so it shows
// the block and lets it be read.
func (divergence Divergence) Lines() []string {
	return append(append([]string{}, divergence.Context...), divergence.Received)
}

// Outcome is how one run ended. Failure is nil when the run passed, and also
// when it could not be started at all -- a missing executable or an unreadable
// tape is a fault in the invocation rather than a divergence, and saying a
// port differs from the reference when it never ran would be a lie of exactly
// the kind this tool exists to catch.
type Outcome struct {
	Name     string
	ExitCode int
	Failure  *Divergence
}

// Report explains a divergence to whoever is watching, in a form meant to be
// read by a person or pasted into a bug report.
//
// Everything in it is either a fixed string, something the script says, or
// something the program printed. Nothing is measured: a timeout reports the
// limit it was given and not how long it actually waited, so two runs of the
// same divergence produce the same bytes and a real change stands out from a
// slow machine.
//
// Lines are quoted because trailing whitespace is load-bearing: prompts
// routinely end in a space, and an unquoted report makes a step that differs
// from another only in that respect look identical to it.
func (divergence Divergence) Report() string {
	if divergence.Matched() {
		return fmt.Sprintf("%s: all %d steps matched, but %s",
			divergence.Mode, divergence.StepCount, divergence.Detail)
	}

	var report strings.Builder
	fmt.Fprintf(&report, "%s: step %d of %d did not match (%s)\n\n",
		divergence.Mode, divergence.StepIndex+1, divergence.StepCount, divergence.Detail)
	if divergence.Redacted {
		fmt.Fprintf(&report, "  expected  %q  (with redactions applied)\n", divergence.Expected)
	} else {
		fmt.Fprintf(&report, "  expected  %q\n", divergence.Expected)
	}
	fmt.Fprintf(&report, "  received  %q\n", divergence.Received)
	// A character-level pointer only means something when the two are meant to
	// be equal; against a pattern the first differing byte is noise.
	if !divergence.Redacted {
		report.WriteString(differenceMarker(divergence.Expected, divergence.Received, len(reportLabelIndent)))
	}

	// The preceding lines are what makes a mismatch diagnosable when the
	// received line is not the interesting one, e.g. the executable printed an
	// error and exited before ever reaching the prompt.
	if len(divergence.Context) > 0 {
		report.WriteString("\n  output since the last matched step:\n")
		for _, line := range divergence.Context {
			fmt.Fprintf(&report, "    %s\n", line)
		}
	}

	if remaining := divergence.StepCount - divergence.StepIndex - 1; remaining > 0 {
		fmt.Fprintf(&report, "\n%s never reached\n", pluralSteps(remaining))
	}

	return report.String()
}

// reportLabelIndent is the prefix in front of the quoted expected/received
// lines. differenceMarker needs its width to line a caret up underneath.
const reportLabelIndent = "  received  "

// differenceMarker points at the first character where the received line
// diverged from the expected one. Spotting that by eye in two quoted strings
// is exactly the tedium this tool exists to remove, and it is worst in the
// cases that matter most: a changed letter mid-word, or a trailing space.
func differenceMarker(expected, received string, indent int) string {
	// With nothing received there is nothing to point at, and the empty quotes
	// above already say it plainly.
	if expected == received || received == "" {
		return ""
	}

	prefix := commonPrefix(expected, received)
	label := "first difference"
	if prefix == received {
		label = "received output ends here"
	}

	// The caret indexes the quoted rendering, so measure the prefix as it will
	// appear once escaped. Quote adds a pair of quotes; only the opening one
	// sits between the indent and the first differing character.
	column := indent + len(strconv.Quote(prefix)) - 1
	return fmt.Sprintf("%s^ %s\n", strings.Repeat(" ", column), label)
}

func commonPrefix(a, b string) string {
	offset := 0
	for _, char := range a {
		width := utf8.RuneLen(char)
		if offset+width > len(b) || a[offset:offset+width] != b[offset:offset+width] {
			break
		}
		offset += width
	}
	return a[:offset]
}

func pluralSteps(count int) string {
	if count == 1 {
		return "1 later step was"
	}
	return fmt.Sprintf("%d later steps were", count)
}
