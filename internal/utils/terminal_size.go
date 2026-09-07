package utils

// The size every one of these programs was written for, and the size the pty
// is given whatever terminal prescript itself was started from.
//
// A pty opened without a size starts at zero rows and columns, so a program
// that asks how wide its terminal is -- to decide how many columns of a table
// to print, or where to wrap -- gets an answer that is neither what a real
// terminal would give it nor what the next machine's terminal would. Either
// way that is a difference in output with no difference in behaviour behind
// it, which is exactly the kind of noise a comparison run has to be free of.
const (
	terminalColumns = 80
	terminalRows    = 24
)

// TerminalSize is the size prescript gives every child's pty, in columns and
// rows. Exported because a bug report has to name it: a program that wraps its
// output at the terminal's width prints something different at a different
// width, and a reader who cannot see which width was used cannot tell that
// from a real difference.
func TerminalSize() (int, int) {
	return terminalColumns, terminalRows
}
