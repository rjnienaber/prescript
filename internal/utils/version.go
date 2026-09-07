package utils

import (
	"runtime/debug"
)

// version can be stamped in at build time with
// -ldflags "-X github.com/rjnienaber/prescript/internal/utils.version=1.2.3".
// Left empty by an ordinary build, which falls back to what the toolchain
// records for itself.
var version = ""

// Version identifies the prescript that produced a report.
//
// A bug report that does not say which build made it cannot be answered: the
// first question about any cross-implementation report is whether the tool
// still behaves that way, and the second is which tool. The Go toolchain
// stamps the commit and whether the tree was dirty into every binary built
// from a repository, so this is available without a release process to
// maintain -- and a build made from uncommitted changes says so, which is the
// case most worth knowing about.
func Version() string {
	if version != "" {
		return version
	}

	info, available := debug.ReadBuildInfo()
	if !available {
		return "unknown"
	}

	revision := ""
	modified := false
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}

	if revision == "" {
		if info.Main.Version != "" {
			return info.Main.Version
		}
		return "unknown"
	}

	if len(revision) > 12 {
		revision = revision[:12]
	}
	if modified {
		return revision + " (modified)"
	}
	return revision
}
