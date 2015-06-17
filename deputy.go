// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

// package deputy provides more advanced options for running commands.
package deputy

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

// ErrorHandling is a flag that tells Deputy how to handle errors running a
// command.
type ErrorHandling int

const (
	// StdErrs represents the default handling of command errors - this simply
	// returns the error from Cmd.Run()
	StdErrs ErrorHandling = iota
	// FromStderr tells Deputy to convert the stderr output of a command into
	// the text of an error, if the command exits with an error.
	FromStderr
	// FromStdout tells Deputy to convert the stdout output of a command into
	// the text of an error, if the command exits with an error.
	FromStdout
)

// Deputy is a type that runs Commands with advanced options not available from
// os/exec.
type Deputy struct {
	Timeout time.Duration
	Errors  ErrorHandling
}

// Run starts the specified command and waits for it to complete.  Its behavior
// conforms to the Options passed to it at construction time.
func (r Deputy) Run(cmd *exec.Cmd) error {
	errsrc := &bytes.Buffer{}
	if r.Errors == FromStderr {
		if cmd.Stderr == nil {
			cmd.Stderr = errsrc
		} else {
			cmd.Stderr = io.MultiWriter(cmd.Stderr, errsrc)
		}
	}
	if r.Errors == FromStdout {
		if cmd.Stdout == nil {
			cmd.Stdout = errsrc
		} else {
			cmd.Stdout = io.MultiWriter(cmd.Stdout, errsrc)
		}
	}

	err := runTimeout(cmd, r.Timeout)

	if r.Errors == StdErrs {
		return err
	}

	if err != nil && errsrc.Len() > 0 {
		return fmt.Errorf("%s: %s", err, strings.TrimSpace(errsrc.String()))
	}
	return err
}

func runTimeout(cmd *exec.Cmd, timeout time.Duration) error {
	if timeout == 0 {
		return cmd.Run()
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	done := make(chan error)

	var err error
	go func() {
		err = cmd.Wait()
		close(done)
	}()

	select {
	case <-time.After(timeout):
		// this may fail, but there's not much we can do about it
		_ = cmd.Process.Kill()
		return timeoutErr{cmd.Path}
	case <-done:
		return err
	}
}

type timeoutErr struct {
	path string
}

func (t timeoutErr) IsTimeout() bool {
	return true
}

func (t timeoutErr) Error() string {
	return fmt.Sprintf("timed out waiting for command %q to execute", t.path)
}
