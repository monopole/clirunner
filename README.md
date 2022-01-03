# clirunner [![Go Report Card](https://goreportcard.com/badge/github.com/monopole/clirunner)](https://goreportcard.com/report/github.com/monopole/clirunner) [![Go Reference](https://pkg.go.dev/badge/github.com/monopole/clirunner)](https://pkg.go.dev/github.com/monopole/clirunner)

Package clirunner runs a legacy shell-style command-line interpreter
as if a human were running it.

Such a CLI typically offers a prompt and a set of commands.  A human
fires it up, issues a command, then reads the output.  Armed with
this new knowledge, the human can issue a new command as a function
of the previous commands output, etc.

The framework here runs the CLI as a subprocess, and arranges for piping
of `stderr` and `stdout` to instances of `ifc.Commander`.

```Go
type Commander interface {
  // Stringer provides a command via String(), e.g. the command "echo".
  fmt.Stringer

  // Writer accepts CLI output for parsing.
  //
  // Output from the CLI subprocess' stdout and stderr should be sent into
  // Write.  Commander can accumulate whatever good data and errors it desires.
  io.Writer
}
```

Your job as a framework user is implement these instances in Go,
and write a main program that feeds them into `ProcRunner`.

The overall experience should be more enjoyable than trying to hack
it all together in bash.
