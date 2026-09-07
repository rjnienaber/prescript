package play

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	cfg "github.com/rjnienaber/prescript/internal/config"
	"github.com/rjnienaber/prescript/internal/script"
	"github.com/rjnienaber/prescript/internal/utils"
)

// RunAll plays every run in a script and returns the first non-success exit
// code, or SUCCESS if they all passed.
//
// Every run is played even after one has failed. A script with more than one
// run exists to compare implementations against each other, and stopping at the
// first failure would withhold exactly the comparison it was asked for: knowing
// that the first of three ports is wrong says nothing about the other two.
func RunAll(config cfg.PlayConfig, runs []script.Run, logger utils.Logger) int {
	multiple := len(runs) > 1
	if multiple {
		warnAboutOverrides(config, len(runs))
	}

	result := utils.SUCCESS
	var failed []string

	for _, run := range runs {
		if multiple {
			// stderr, not stdout: stdout stays the program's own output, so a
			// single-run script that is piped somewhere is unaffected by this,
			// and --quiet still silences the program without hiding which run
			// a failure report belongs to.
			fmt.Fprintf(os.Stderr, "\n=== %s ===\n", run.Name)
		}

		code := Run(config, run, logger)
		if code == utils.SUCCESS {
			continue
		}

		failed = append(failed, strconv.Quote(run.Name))
		if result == utils.SUCCESS {
			result = code
		}
	}

	if multiple && len(failed) > 0 {
		fmt.Fprintf(os.Stderr, "\n%d of %d runs failed: %s\n", len(failed), len(runs), strings.Join(failed, ", "))
	}

	return result
}

// warnAboutOverrides is a warning rather than a refusal. Overriding one thing
// for every run is meaningful when the runs differ in their steps, and a
// mistake when they differ in their executable, and prescript cannot tell which
// was meant. Saying so is cheaper than being wrong in either direction.
func warnAboutOverrides(config cfg.PlayConfig, runCount int) {
	if config.ExecutablePath != "" {
		fmt.Fprintf(os.Stderr, "warning: --exec applies to all %d runs, so every one of them will use %s\n", runCount, config.ExecutablePath)
	}

	if len(config.Arguments) > 0 {
		fmt.Fprintf(os.Stderr, "warning: the arguments after -- replace those of all %d runs\n", runCount)
	}
}
