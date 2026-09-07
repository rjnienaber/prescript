//go:build !unix

package utils

import (
	"errors"
	"os"
	"os/exec"
)

// startPty reports that this platform has no pseudo-terminal to offer. The
// error names the way out rather than only the problem, because a script that
// says nothing about terminals is asking for the default and has no idea it is
// about to be unavailable.
func startPty(cmd *exec.Cmd) (*os.File, error) {
	return nil, errors.New("a pty is not supported on this platform; run with --terminal pipes")
}

// killGroup has nothing to sign off on where there is no pty to have made a
// process group in the first place.
func killGroup(pid int) error {
	return errors.New("killing a process group is not supported on this platform")
}
