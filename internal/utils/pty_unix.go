//go:build unix

package utils

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"
)

// startPty starts cmd with a pseudo-terminal on all three of its standard
// streams and returns this process's end of it.
//
// The child's end is put into raw mode before the child is started, so there
// is no window in which the terminal's line discipline can touch its first
// line of output. Two of the flags raw mode clears are the ones that matter
// here:
//
//   - ECHO would send every character prescript types back down the same
//     stream it is matching against, so a script would have to expect its own
//     input interleaved with the program's output.
//   - ONLCR would turn each "\n" the program writes into "\r\n".
//
// What is left is a stream of bytes identical to what a pipe would have
// delivered, from a program that can nonetheless see it is talking to a
// terminal. That is the whole point: isatty() is what a C or Python program
// consults to decide whether to line-buffer its output or block-buffer it, and
// under a pipe a prompt with no trailing newline can sit in a buffer forever
// while prescript waits for it.
func startPty(cmd *exec.Cmd) (*os.File, error) {
	ptmx, tty, err := pty.Open()
	if err != nil {
		return nil, err
	}
	// The child gets its own descriptor for the tty at Start. This process
	// must not keep one open, or reads from ptmx would never report the child
	// having gone away.
	defer func() { _ = tty.Close() }()

	closePty := func(err error) (*os.File, error) {
		_ = ptmx.Close()
		return nil, err
	}

	if _, err := term.MakeRaw(int(tty.Fd())); err != nil {
		return closePty(err)
	}

	cmd.Stdin, cmd.Stdout, cmd.Stderr = tty, tty, tty

	// Setsid and Setctty make the child a session leader with this pty as its
	// controlling terminal, so a program that opens /dev/tty explicitly —
	// which is how a password prompt avoids a redirected stdin — finds this
	// one. Ctty is an index into the child's descriptors, and stdin is 0.
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setsid = true
	cmd.SysProcAttr.Setctty = true
	cmd.SysProcAttr.Ctty = 0

	if err := cmd.Start(); err != nil {
		return closePty(err)
	}

	return ptmx, nil
}
