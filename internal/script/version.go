package script

import (
	"fmt"
	"strconv"
	"strings"
)

// knownVersions lists every script format version this build can read, oldest
// first. The last entry is what `record` writes.
//
// The format is MAJOR.MINOR:
//
//   - MINOR increases when the change is additive, for example a new optional
//     field. Scripts written against an earlier minor keep working unchanged,
//     so there is nothing to migrate.
//   - MAJOR increases when existing scripts would have to be rewritten. Older
//     majors are dropped from this list at that point, and the error names the
//     last release that could read them.
//
// Add a version here in the same change that alters the schema, never before.
var knownVersions = []string{"0.1", "0.2", "0.3", "0.4", "0.5", "0.6", "0.7", "0.8"}

// CurrentVersion is the version `record` writes and the newest this build
// understands.
var CurrentVersion = knownVersions[len(knownVersions)-1]

// checkVersion decides whether this build can read a script.
//
// An unrecognised version is rejected rather than warned about, and that is the
// important half of the rule. Additive fields are exactly the ones that change
// what a run means -- an environment variable, a redaction rule, a second run
// to compare against -- so a reader that skipped what it did not recognise
// would not fail, it would quietly do something else and report success. A
// wrong answer delivered confidently is the failure this tool exists to catch,
// so it must not be the failure the tool itself commits.
func checkVersion(declared string) error {
	for _, known := range knownVersions {
		if declared == known {
			return nil
		}
	}

	if newer, err := isNewerThanCurrent(declared); err == nil && newer {
		return fmt.Errorf(
			"version: script is written for format %s, but this build of prescript reads up to %s; upgrade prescript",
			declared, CurrentVersion)
	}

	return fmt.Errorf("version: unrecognised script format %q: expected one of %s",
		declared, strings.Join(quoteAll(knownVersions), ", "))
}

func isNewerThanCurrent(declared string) (bool, error) {
	declaredMajor, declaredMinor, err := splitVersion(declared)
	if err != nil {
		return false, err
	}

	currentMajor, currentMinor, err := splitVersion(CurrentVersion)
	if err != nil {
		return false, err
	}

	if declaredMajor != currentMajor {
		return declaredMajor > currentMajor, nil
	}
	return declaredMinor > currentMinor, nil
}

func splitVersion(version string) (int, int, error) {
	parts := strings.Split(version, ".")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("version %q is not MAJOR.MINOR", version)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}

	return major, minor, nil
}

func quoteAll(values []string) []string {
	quoted := make([]string, len(values))
	for i, value := range values {
		quoted[i] = strconv.Quote(value)
	}
	return quoted
}
