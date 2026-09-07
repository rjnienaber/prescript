package script

import (
	"regexp"
	"time"
)

type Step struct {
	Line      string        `json:"line"`
	LineRegex regexp.Regexp `json:"-"`
	Input     string        `json:"input,omitempty"`
	IsRegex   bool          `json:"isRegex,omitempty"`

	// Redacted is set while parsing, for a line that referred to a redaction
	// and so matches by pattern rather than by equality. It is not part of the
	// file format: the placeholders in Line are.
	Redacted bool `json:"-"`
}

// UsesPattern reports whether the step matches by regular expression rather
// than by exact equality, whichever of the two ways it got there.
func (step Step) UsesPattern() bool {
	return step.IsRegex || step.Redacted
}

type Run struct {
	Name       string    `json:"name,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
	Executable string    `json:"executable"`
	Arguments  []string  `json:"arguments"`
	ExitCode   int       `json:"exitCode"`
	Steps      []Step    `json:"steps"`

	// Env is the environment the run declares for its child process, and
	// InheritEnv asks for prescript's own environment underneath it. See
	// Run.Environment for what an absent, empty or populated Env means.
	Env        map[string]string `json:"env,omitempty"`
	InheritEnv bool              `json:"inheritEnv,omitempty"`

	// RunnerArguments come from a runner file rather than from the script, and
	// always precede the run's own arguments: the runner names the interpreter
	// and its flags, the script names the program to feed it.
	RunnerArguments []string `json:"-"`
}

type Script struct {
	Version string `json:"version"`

	// Redactions are named patterns that stand in for the parts of a line that
	// vary between runs, referred to from a step's line as {{name}}. They are
	// declared once for the whole script so that two runs being compared
	// against each other normalise the same things in the same way.
	Redactions map[string]string `json:"redactions,omitempty"`

	Runs []Run `json:"runs"`
}
