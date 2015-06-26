// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package deputy

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
)

type suite struct{}

var _ = gc.Suite(&suite{})

func Test(t *testing.T) {
	gc.TestingT(t)
}

type hasTimeout interface {
	IsTimeout() bool
}

func (*suite) TestRunTimeout(c *gc.C) {
	cmd := maker{
		timeout: time.Second * 2,
	}.make()

	err := Deputy{Timeout: time.Millisecond * 100}.Run(cmd)

	c.Assert(err, gc.NotNil)
	if e, ok := err.(hasTimeout); !ok {
		c.Errorf("Error caused by timeout does not have Timeout function")
	} else {
		c.Assert(e.IsTimeout(), jc.IsTrue)
	}
}

func (*suite) TestRunNoTimeout(c *gc.C) {
	cmd := maker{}.make()

	err := Deputy{Timeout: time.Millisecond * 200}.Run(cmd)

	c.Assert(err, gc.IsNil)
}

func (*suite) TestStdoutErr(c *gc.C) {
	output := "foooo"
	cmd := maker{
		stdout: output,
		exit:   1,
	}.make()
	err := Deputy{Errors: FromStdout}.Run(cmd)
	c.Assert(err, gc.ErrorMatches, ".*"+output)
}

func (*suite) TestStdoutOutput(c *gc.C) {
	output := "foooo"
	out := &bytes.Buffer{}
	cmd := maker{
		stdout: output,
		exit:   1,
	}.make()
	cmd.Stdout = out
	err := Deputy{Errors: FromStdout}.Run(cmd)
	c.Check(err, gc.ErrorMatches, ".*"+output)
	c.Check(output, gc.Equals, strings.TrimSpace(out.String()))
}

func (*suite) TestStderrOutput(c *gc.C) {
	output := "foooo"
	out := &bytes.Buffer{}

	cmd := maker{
		stderr: output,
		exit:   1,
	}.make()
	cmd.Stderr = out
	err := Deputy{Errors: FromStderr}.Run(cmd)
	c.Assert(err, gc.ErrorMatches, ".*"+output)
	c.Assert(output, gc.Equals, strings.TrimSpace(out.String()))
}

func (*suite) TestStderrErr(c *gc.C) {
	output := "foooo"

	cmd := maker{
		stderr: output,
		exit:   1,
	}.make()
	err := Deputy{Errors: FromStderr}.Run(cmd)
	c.Assert(err, gc.ErrorMatches, ".*"+output)
}

func (*suite) TestLogs(c *gc.C) {
	stdout := "foo!"
	stderr := "bar!"
	cmd := maker{
		stderr: stderr,
		stdout: stdout,
	}.make()
	var logout []byte
	var logerr []byte

	err := Deputy{
		StdoutLog: func(b []byte) { logout = b },
		StderrLog: func(b []byte) { logerr = b },
	}.Run(cmd)
	c.Assert(err, jc.ErrorIsNil)
	c.Check(string(logout), gc.DeepEquals, stdout)
	c.Check(string(logerr), gc.DeepEquals, stderr)
}

type maker struct {
	stdout  string
	stderr  string
	exit    int
	timeout time.Duration
}

const (
	isHelperProc  = "GO_HELPER_PROCESS_OK"
	helperStdout  = "GO_HELPER_PROCESS_STDOUT"
	helperStderr  = "GO_HELPER_PROCESS_STDERR"
	helperExit    = "GO_HELPER_PROCESS_EXIT_CODE"
	helperTimeout = "GO_HELPER_PROCESS_TIMEOUT"
)

func (m maker) make() *exec.Cmd {
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess")
	cmd.Env = []string{
		fmt.Sprintf("%s=%s", isHelperProc, "1"),
		fmt.Sprintf("%s=%s", helperStdout, m.stdout),
		fmt.Sprintf("%s=%s", helperStderr, m.stderr),
		fmt.Sprintf("%s=%d", helperExit, m.exit),
		fmt.Sprintf("%s=%d", helperTimeout, m.timeout.Nanoseconds()),
	}
	return cmd
}

func TestHelperProcess(*testing.T) {
	if os.Getenv(isHelperProc) != "1" {
		return
	}
	exit, err := strconv.Atoi(os.Getenv(helperExit))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error converting exit code: %s", err)
		os.Exit(2)
	}
	defer os.Exit(exit)

	nanos, err := strconv.Atoi(os.Getenv(helperTimeout))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error converting timeout: %s", err)
		os.Exit(2)
	}
	<-time.After(time.Duration(int64(nanos)) * time.Nanosecond)
	if stderr := os.Getenv(helperStderr); stderr != "" {
		fmt.Fprint(os.Stderr, stderr)
	}
	if stdout := os.Getenv(helperStdout); stdout != "" {
		fmt.Fprint(os.Stdout, stdout)
	}
}
