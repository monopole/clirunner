// Package clirunner runs a shell-style CLI as if a human were running it.
//
// Use one instance of ProcRunner to manage the CLI (kubectl, mysql, etc.),
// and for each different command you want to run you need an implementation
// of ifc.Commander.
//
// The Commander implementation knows both the command string and how to parse
// the command's output from the CLI.  ProcRunner takes care of all the wiring
// and deadline monitoring.
//
// See example_test.go for an example, and ProcRunner and Commander for detailed
// documentation.
package clirunner
