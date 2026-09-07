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
}

type Script struct {
	Version string `json:"version"`
	Runs    []Run  `json:"runs"`
}
