package deputy_test

import (
	"log"
	"os/exec"
	"time"

	"npf.io/deputy"
)

func Example() {
	cancel := make(chan struct{})
	go func() {
		<-time.After(time.Second * 30)
		close(cancel)
	}()

	// Make a new deputy that'll return the data written to stderr as the error
	// message, log everything written to stdout to this application's log,  and
	// timeout after 30 seconds.
	d := deputy.Deputy{
		Errors:    deputy.FromStderr,
		StdoutLog: func(b []byte) { log.Print(string(b)) },
		Cancel:    cancel,
	}
	if err := d.Run(exec.Command("foo")); err != nil {
		log.Print(err)
	}
}
