package script

import "sort"

// Environment returns the environment for a run's child process, or nil to mean
// "inherit prescript's own", which is what exec.Cmd does with a nil Env.
//
// A run that declares no env inherits, as prescript has always done. A run that
// declares one gets exactly what it declared and nothing else, unless it also
// sets inheritEnv. Declaring an environment is how a script says the run does
// not depend on whatever happened to be exported around it; quietly folding in
// the caller's LANG, TZ and PATH would take that guarantee away, and those are
// the variables most likely to change a program's output between two machines.
func (run Run) Environment(parent []string) []string {
	if run.Env == nil {
		return nil
	}

	// Not nil: an explicit "env": {} with no inheritance is an empty
	// environment, which is a different instruction from "inherit".
	env := []string{}
	if run.InheritEnv {
		env = append(env, parent...)
	}

	// Sorted so a run builds the same environment every time. exec keeps the
	// last assignment to a name, so declared values override inherited ones.
	names := make([]string, 0, len(run.Env))
	for name := range run.Env {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		env = append(env, name+"="+run.Env[name])
	}
	return env
}
