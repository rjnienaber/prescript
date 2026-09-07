package script

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// redactionPlaceholder finds a {{name}} reference in an expected line. Two
// braces because one is not rare enough: program output contains { often and
// {{ essentially never.
var redactionPlaceholder = regexp.MustCompile(`\{\{([^{}]*)\}\}`)

// compileRedactions turns every step line containing a {{name}} placeholder
// into a regular expression, substituting the named pattern and quoting
// everything around it.
//
// This is what a fixture wants instead of isRegex. Masking one volatile number
// with isRegex means hand-writing a regular expression for the whole line, at
// which point the line stops being readable as the output it stands for and
// the intent behind the mask is lost. A name keeps both: the line still looks
// like what the program prints, and {{seed}} says why that part varies.
func compileRedactions(script Script) []string {
	var errors []string

	known := map[string]bool{}
	for _, name := range sortedRedactionNames(script.Redactions) {
		if _, err := regexp.Compile(script.Redactions[name]); err != nil {
			errors = append(errors, fmt.Sprintf("redactions.%s: %s", name, err))
			continue
		}
		known[name] = true
	}

	for runIndex, run := range script.Runs {
		for stepIndex := range run.Steps {
			step := &run.Steps[stepIndex]
			placeholders := redactionPlaceholder.FindAllStringSubmatchIndex(step.Line, -1)
			if len(placeholders) == 0 {
				continue
			}

			where := fmt.Sprintf("runs.%d.steps.%d.line", runIndex, stepIndex)
			if step.IsRegex {
				errors = append(errors, where+": a step cannot both be a regular expression and use redactions")
				continue
			}

			pattern, unknown := buildPattern(step.Line, placeholders, script.Redactions, known)
			if len(unknown) > 0 {
				errors = append(errors, fmt.Sprintf("%s: %s", where, unknownRedactions(unknown, script.Redactions)))
				continue
			}

			compiled, err := regexp.Compile(pattern)
			if err != nil {
				// Every part came from a pattern that compiled on its own, so
				// this is the combination failing, not one of the pieces.
				errors = append(errors, fmt.Sprintf("%s: redactions do not combine into a usable pattern: %s", where, err))
				continue
			}

			step.LineRegex = *compiled
			step.Redacted = true
		}
	}

	return errors
}

// buildPattern anchors at both ends: a redacted line stands for the whole line
// the program printed, the same as a line matched literally does. isRegex is
// unanchored and stays that way, because scripts already rely on it.
func buildPattern(line string, placeholders [][]int, redactions map[string]string, known map[string]bool) (string, []string) {
	var pattern strings.Builder
	var unknown []string

	pattern.WriteString("^")
	end := 0

	for _, placeholder := range placeholders {
		name := line[placeholder[2]:placeholder[3]]
		pattern.WriteString(regexp.QuoteMeta(line[end:placeholder[0]]))
		end = placeholder[1]

		if !known[name] {
			// A name whose pattern failed to compile is reported against the
			// redaction itself; repeating it against every step that uses it
			// would bury the one line worth fixing.
			if _, declared := redactions[name]; !declared {
				unknown = append(unknown, name)
			}
			continue
		}

		pattern.WriteString("(?:" + redactions[name] + ")")
	}

	pattern.WriteString(regexp.QuoteMeta(line[end:]))
	pattern.WriteString("$")

	return pattern.String(), unknown
}

func unknownRedactions(unknown []string, redactions map[string]string) string {
	subject := fmt.Sprintf("unknown redaction %s", strconv.Quote(unknown[0]))
	if len(unknown) > 1 {
		subject = "unknown redactions " + strings.Join(quoteAll(unknown), ", ")
	}

	declared := sortedRedactionNames(redactions)
	if len(declared) == 0 {
		return subject + "; no redactions are declared in this script"
	}
	return subject + "; declared: " + strings.Join(quoteAll(declared), ", ")
}

func sortedRedactionNames(redactions map[string]string) []string {
	names := make([]string, 0, len(redactions))
	for name := range redactions {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
