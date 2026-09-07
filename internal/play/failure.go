package play

// FailureMode names why a run stopped, from a fixed set. It leads every
// failure report.
//
// The three ways a run can end short are three different bugs — the program
// printed something else, the program died, the program is waiting for
// something nobody is going to send — and told apart they point at different
// places to look. Told apart by a stable token rather than by prose, a corpus
// of a few hundred runs can be counted and sorted by what went wrong, and the
// same divergence reads the same way every time it happens.
type FailureMode string

const (
	// NoMatch: the program is still running, and what it is printing is not
	// what the step expects. Usually the implementation prints something
	// different from the reference.
	NoMatch FailureMode = "no-match"

	// ExitedEarly: the program finished while steps were still outstanding.
	// Usually it crashed, or refused an argument, or the script outlasts it.
	ExitedEarly FailureMode = "exited-early"

	// Hung: every step matched and the program never exited. Usually it is
	// waiting for input the script does not go on to supply.
	Hung FailureMode = "hung"

	// WrongExitCode: it ran to the end, printed everything expected of it, and
	// disagreed only about how it finished.
	WrongExitCode FailureMode = "wrong-exit-code"

	// ReadFailed: prescript could not read from the program at all, which is a
	// fault in prescript or in the machine rather than in the program.
	ReadFailed FailureMode = "read-failed"
)
