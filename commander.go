package clirunner

import (
	"fmt"
	"io"
)

// Commander knows a CLI command, and knows how to parse the command's output.
type Commander interface {
	// Stringer provides a command via String(), e.g. the command "echo".
	fmt.Stringer

	// Writer accepts CLI output for parsing.
	//
	// Output from the CLI subprocess' stdout and stderr should be sent into
	// Write.  Commander can accumulate whatever good data and errors it desires.
	//
	// A commander should return an error from Write only on some sort of
	// catastrophe, as the error will result in shutting down the CLI subprocess.
	// An implementation may choose to allow certain unexpected CLI output and
	// not return an error, instead merely counting and/or logging the error.
	io.Writer

	// Success returns true if the Commander decided that it succeeded
	// in parsing output from the CLI.  Doesn't necessarily imply that
	// the command ran without error.  This value can be consulted by whatever
	// program is coordinating the ProcRunner and Commander instances.
	Success() bool

	// Reset resets the internal state of the parser (e.g. error counts)
	// and sets Success to false.  This allows the Commander instance to be
	// used in another Run.
	Reset()
}
