package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"

	cli2 "github.com/monopole/clirunner/internal/testcli/tstcli"
)

//go:embed README.md
var readMeMd string

type argSack struct {
	numRowsInDb   int
	rowToErrorOn  int
	disablePrompt bool
	exitOnError   bool
}

// main reads commands from stdin, pretending to be a database frontend CLI.
func main() {
	var args argSack
	flag.IntVar(
		&args.rowToErrorOn,
		cli2.FlagRowToErrorOn, 0,
		"Error if this row number is in the results.")
	flag.IntVar(
		&args.numRowsInDb,
		cli2.FlagNumRowsInDb, 50,
		"Maximum number of rows in the fake db.")
	flag.BoolVar(
		&args.disablePrompt,
		cli2.FlagDisablePrompt, false,
		"Disable the prompt.")
	flag.BoolVar(
		&args.exitOnError,
		cli2.FlagExitOnErr, false,
		"Exit on error, else continue accepting commands.")
	flag.Parse()
	if len(flag.Args()) > 0 {
		if flag.Args()[0] != cli2.CmdHelp {
			fmt.Fprintln(os.Stderr, "unrecognized args: ", flag.Args())
		}
		fmt.Fprintln(os.Stderr)
		fmt.Fprint(os.Stderr, readMeMd)
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "Commands: %v\n", cli2.AllCommands)
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		os.Exit(1)
	}
	shell := cli2.NewShell(
		cli2.NewSillyDb(args.numRowsInDb, args.rowToErrorOn),
		args.disablePrompt,
		args.exitOnError,
		readMeMd,
	)
	if err := shell.Run(); err != nil {
		// Assume error was already printed.
		os.Exit(1)
	}
}
