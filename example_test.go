// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package deputy_test

import (
	"log"
	"os/exec"
	"time"

	"github.com/juju/deputy"
)

func Example() {
	// Make a new deputy that'll return the data written to stderr as the error
	// message, log everything written to stdout to this application's log,  and
	// timeout after 30 seconds.
	d := deputy.Deputy{
		Errors:    deputy.FromStderr,
		StdoutLog: log.Print,
		Timeout:   time.Second * 30,
	}
	if err := d.Run(exec.Command("foo")); err != nil {
		log.Print(err)
	}
}
