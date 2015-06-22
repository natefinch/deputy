# deputy [![GoDoc](https://godoc.org/github.com/juju/deputy?status.svg)](https://godoc.org/github.com/juju/deputy)
deputy is a go package that adds smarts on top of os/exec

![deputy-sm](https://cloud.githubusercontent.com/assets/3185864/8237448/6bc30102-15bd-11e5-9e87-6423197a73d6.jpg)

<sup><sub>image: creative commons, &copy; [MatsuRD](http://matsurd.deviantart.com/art/Paper53-Deputy-Stubbs-342123485)</sub></sup>

## Example

``` go
// Make a new deputy that'll return the data written to stderr as the error
// message, log everything written to stdout to this application's log,  and
// timeout after 30 seconds.
d := deputy.Deputy{
    Errors:    deputy.FromStderr,
    StdoutLog: func(s string) { log.Print(s) },
    Timeout:   time.Second * 30,
}
if err := d.Run(exec.Command("foo")); err != nil {
    log.Print(err)
}
```

## type Deputy
``` go
type Deputy struct {
    // Timeout represents the longest time the command will be allowed to run
    // before being killed.
    Timeout time.Duration
    // Errors describes how errors should be handled.
    Errors ErrorHandling
    // StdoutLog takes a function that will receive lines written to stdout from
    // the command.
    StdoutLog func(string)
    // StdoutLog takes a function that will receive lines written to stderr from
    // the command.
    StderrLog func(string)
    // contains filtered or unexported fields
}
```
Deputy is a type that runs Commands with advanced options not available from
os/exec.  See the comments on field values for details.

### func (Deputy) Run
``` go
func (d Deputy) Run(cmd *exec.Cmd) error
```
Run starts the specified command and waits for it to complete.  Its behavior
conforms to the Options passed to it at construction time.

## type ErrorHandling
``` go
type ErrorHandling int
```
ErrorHandling is a flag that tells Deputy how to handle errors running a
command.  See the values below for the different modes.

``` go
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
```