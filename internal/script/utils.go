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
	Timestamp  time.Time `json:"timestamp"`
	Executable string    `json:"executable"`
	Arguments  []string  `json:"arguments"`
	ExitCode   int       `json:"exitCode"`
	Steps      []Step    `json:"steps"`
}

type Script struct {
	Version string `json:"version"`
	Runs    []Run  `json:"runs"`
}
