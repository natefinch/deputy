// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

// Package deputy provides more advanced options for running commands.
package deputy

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"time"
)

// ErrorHandling is a flag that tells Deputy how to handle errors running a
// command.  See the values below for the different modes.
type ErrorHandling int

const (
	// DefaultErrs represents the default handling of command errors - this
	// simply returns the error from Cmd.Run()
	DefaultErrs ErrorHandling = iota

	// FromStderr tells Deputy to convert the stderr output of a command into
	// the text of an error, if the command exits with an error.
	FromStderr

	// FromStdout tells Deputy to convert the stdout output of a command into
	// the text of an error, if the command exits with an error.
	FromStdout
)

// Deputy is a type that runs Commands with advanced options not available from
// os/exec.  See the comments on field values for details.
type Deputy struct {
	// Timeout represents the longest time the command will be allowed to run
	// before being killed.
	Timeout time.Duration
	// Errors describes how errors should be handled.
	Errors ErrorHandling
	// StdoutLog takes a function that will receive lines written to stdout from
	// the command (with the newline elided).
	StdoutLog func([]byte)
	// StdoutLog takes a function that will receive lines written to stderr from
	// the command (with the newline elided).
	StderrLog func([]byte)

	stderrPipe io.ReadCloser
	stdoutPipe io.ReadCloser
}

// Deputyer is an interface for the Deputy struct
// Added to make it easier to mock out Deputy in unit tests
type Deputyer interface {
	Run(cmd *exec.Cmd) error
}

// Checking that the Deputy struct implements the interface
var _ Deputyer = (*Deputy)(nil)

// Run starts the specified command and waits for it to complete.  Its behavior
// conforms to the Options passed to it at construction time.
//
// Note that, like cmd.Run, Deputy.Run should not be used with
// StdoutPipe or StderrPipe.
func (d Deputy) Run(cmd *exec.Cmd) error {
	if err := d.makePipes(cmd); err != nil {
		return err
	}

	errsrc := &bytes.Buffer{}
	if d.Errors == FromStderr {
		cmd.Stderr = dualWriter(cmd.Stderr, errsrc)
	}
	if d.Errors == FromStdout {
		cmd.Stdout = dualWriter(cmd.Stdout, errsrc)
	}

	err := d.run(cmd)

	if d.Errors == DefaultErrs {
		return err
	}

	if err != nil && errsrc.Len() > 0 {
		return fmt.Errorf("%s: %s", err, bytes.TrimSpace(errsrc.Bytes()))
	}
	return err
}

func (d *Deputy) makePipes(cmd *exec.Cmd) error {
	if d.StderrLog != nil {
		var err error
		d.stderrPipe, err = cmd.StderrPipe()
		if err != nil {
			return err
		}
	}
	if d.StdoutLog != nil {
		var err error
		d.stdoutPipe, err = cmd.StdoutPipe()
		if err != nil {
			return err
		}
	}
	return nil
}

func dualWriter(w1, w2 io.Writer) io.Writer {
	if w1 == nil {
		return w2
	}
	if w2 == nil {
		return w1
	}
	return io.MultiWriter(w1, w2)
}

func (d Deputy) run(cmd *exec.Cmd) error {
	errs := make(chan error)
	if err := d.start(cmd, errs); err != nil {
		return err
	}
	if d.Timeout == 0 {
		return d.wait(cmd, errs)
	}

	done := make(chan error)

	var err error
	go func() {
		err = d.wait(cmd, errs)
		close(done)
	}()

	select {
	case <-time.After(d.Timeout):
		// this may fail, but there's not much we can do about it
		_ = cmd.Process.Kill()
		return timeoutErr{cmd.Path}
	case <-done:
		return err
	}
}

func (d Deputy) start(cmd *exec.Cmd, errs chan<- error) error {
	if err := cmd.Start(); err != nil {
		return err
	}

	if d.stdoutPipe != nil {
		go pipe(d.StdoutLog, d.stdoutPipe, errs)
	}
	if d.stderrPipe != nil {
		go pipe(d.StderrLog, d.stderrPipe, errs)
	}
	return nil
}

func (d Deputy) wait(cmd *exec.Cmd, errs <-chan error) error {
	// Note that it's important that we wait for the pipes
	// to be closed before calling cmd.Wait otherwise
	// Wait can close the pipes before we have read
	// all their data.
	var err1, err2 error
	if d.stdoutPipe != nil {
		err1 = <-errs
	}
	if d.stderrPipe != nil {
		err2 = <-errs
	}
	err := cmd.Wait()
	return firstErr(err, err1, err2)
}

func firstErr(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func pipe(log func([]byte), r io.Reader, errs chan<- error) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		b := scanner.Bytes()
		log(b)
	}

	errs <- scanner.Err()
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
