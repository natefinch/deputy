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
)

func TestRunCancel(t *testing.T) {
	cmd := maker{
		timeout: time.Second * 2,
	}.make()

	cancel := make(chan struct{})
	started := make(chan struct{})
	finished := make(chan struct{})
	var err error
	go func() {
		close(started)
		err = Deputy{Cancel: cancel}.Run(cmd)
		close(finished)
	}()
	select {
	case <-started:
	// good!
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for goroutine to run")
	}
	// give the code time to run a little
	time.Sleep(50 * time.Millisecond)
	close(cancel)
	select {
	case <-finished:
	// good!
	case <-time.After(time.Second):
		t.Fatal("goroutine never cancelled!")
	}

	if err != nil {
		t.Fatalf("unexpected error returned from Run: %v", err)
	}
}

func TestRunNoTimeout(t *testing.T) {
	cmd := maker{}.make()
	err := Deputy{}.Run(cmd)

	if err != nil {
		t.Fatalf("unexpected error returned from Run: %v", err)
	}
}

func TestStdoutErr(t *testing.T) {
	output := "foooo"
	cmd := maker{
		stdout: output,
		exit:   1,
	}.make()
	err := Deputy{Errors: FromStdout}.Run(cmd)
	if !strings.HasSuffix(err.Error(), output) {
		t.Fatalf("Expected output of %q but got %q", output, err)
	}
}

func TestStdoutOutput(t *testing.T) {
	output := "foooo"
	out := &bytes.Buffer{}
	cmd := maker{
		stdout: output,
		exit:   1,
	}.make()
	cmd.Stdout = out
	err := Deputy{Errors: FromStdout}.Run(cmd)
	if !strings.HasSuffix(err.Error(), output) {
		t.Fatalf("Expected output of %q but got %q", output, err)
	}
	stdout := strings.TrimSpace(out.String())
	if stdout != output {
		t.Fatalf("Expected stdout of %q but got %q", output, stdout)
	}
}

func TestStderrOutput(t *testing.T) {
	output := "foooo"
	out := &bytes.Buffer{}

	cmd := maker{
		stderr: output,
		exit:   1,
	}.make()
	cmd.Stderr = out
	err := Deputy{Errors: FromStderr}.Run(cmd)
	if !strings.HasSuffix(err.Error(), output) {
		t.Fatalf("Expected output of %q but got %q", output, err)
	}
	stderr := strings.TrimSpace(out.String())
	if stderr != output {
		t.Fatalf("Expected stderr of %q but got %q", output, stderr)
	}
}

func TestStderrErr(t *testing.T) {
	output := "foooo"

	cmd := maker{
		stderr: output,
		exit:   1,
	}.make()
	err := Deputy{Errors: FromStderr}.Run(cmd)
	if !strings.HasSuffix(err.Error(), output) {
		t.Fatalf("Expected output of %q but got %q", output, err)
	}
}

func TestLogs(t *testing.T) {
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
	if err != nil {
		t.Fatalf("unexpected error returned from Run: %v", err)
	}
	if string(logout) != stdout {
		t.Fatalf("expected stdout to be %q but got %q", stdout, logout)
	}
	if string(logerr) != stderr {
		t.Fatalf("expected stder to be %q but got %q", stderr, logerr)
	}
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
