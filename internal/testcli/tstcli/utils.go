package tstcli

import (
	"time"

	"github.com/monopole/clirunner/cmdrs"
	"github.com/monopole/clirunner/ifc"
)

// MakeOutSentinelCommander returns a sentinel commander that has testcli
// echo something unique to the output.
func MakeOutSentinelCommander() *cmdrs.SimpleSentinelCommander {
	return &cmdrs.SimpleSentinelCommander{
		Command: CmdEcho + " Rumpelstiltskin",
		// the response from testcli on stdOut is just the argument.
		Value: "Rumpelstiltskin",
	}
}

// MakeErrSentinelCommander returns a sentinel commander with a command that's
// unknown to the testcli; it triggers an error when used.
func MakeErrSentinelCommander() *cmdrs.SimpleSentinelCommander {
	return &cmdrs.SimpleSentinelCommander{
		// A command not known to testcli.
		Command: "blahblah",
		// The output message from testcli on stdErr complaining about it.
		Value: "unrecognized command: \"blahblah\"",
	}
}

// MakeSleepCommander Returns a command that makes the testcli sleep for the
// given duration. Used in testing timeouts.
func MakeSleepCommander(d time.Duration) ifc.Commander {
	return cmdrs.NewHoardingCommander(CmdSleep + " " + d.String())
}
