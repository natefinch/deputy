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

// Option is a type that encapsulates ways to alter Deputy's standard running of
// commands.
type Option func(*Deputy)

// Timeout returns an Option that tells the Deputy to terminate the command
// after an amount of time has passed.  If terminated this way, the error value
// returned will have a Timeout() bool function that returns true.
func Timeout(d time.Duration) Option {
	return func(r *Deputy) {
		r.timeout = d
	}
}

// StdoutErr returns an Option that tells the Deputy to convert whatever is
// written to stdout into a go error to be returned, if the command does not run
// successfully.  This will be returned instead of the standard exit value
// error.
func StdoutErr() Option {
	return func(r *Deputy) {
		r.stdoutErr = true
	}
}

// StderrErr returns an Option that tells the Deputy to convert whatever is
// written to stderr into a go error to be returned, if the command does not run
// successfully.  This will be returned instead of the standard exit value
// error.
func StderrErr() Option {
	return func(r *Deputy) {
		r.stderrErr = true
	}
}

// New returns a Deputy that runs commands using the given Options.
func New(options ...Option) Deputy {
	r := Deputy{}
	for _, opt := range options {
		opt(&r)
	}
	return r
}

// Deputy is a type that runs Commands with advanced options not available from
// os/exec.
type Deputy struct {
	timeout   time.Duration
	stdoutErr bool
	stderrErr bool
}

// Run starts the specified command and waits for it to complete.  Its behavior
// conforms to the Options passed to it at construction time.
func (r Deputy) Run(cmd *exec.Cmd) error {
	errsrc := &bytes.Buffer{}
	if r.stderrErr {
		if cmd.Stderr == nil {
			cmd.Stderr = errsrc
		} else {
			cmd.Stderr = io.MultiWriter(cmd.Stderr, errsrc)
		}
	}
	if r.stdoutErr {
		if cmd.Stdout == nil {
			cmd.Stdout = errsrc
		} else {
			cmd.Stdout = io.MultiWriter(cmd.Stdout, errsrc)
		}
	}

	err := runTimeout(cmd, r.timeout)

	if !r.stdoutErr && !r.stderrErr {
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
