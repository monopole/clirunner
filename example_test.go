package clirunner_test

import (
	"os"

	. "github.com/monopole/clirunner"
	"github.com/monopole/clirunner/cmdrs"
	cli2 "github.com/monopole/clirunner/internal/testcli/tstcli"
)

func ExampleProcRunner_basicRun() {
	runner, _ := NewProcRunner(&Parameters{
		Path: cli2.TestCliPath,
		Args: []string{
			"--" + cli2.FlagDisablePrompt,
		},
		ExitCommand: cli2.CmdQuit,
		OutSentinel: cli2.MakeOutSentinelCommander(),
	})
	commander := cmdrs.NewPrintingCommander("query limit 3", os.Stdout)
	assertNoErr(runner.RunIt(commander, testingTimeout))

	// Output:
	// Cempedak_|_Bamberga_|_4_|_00000000000000000000000000000001
	// Buddha's hand_|_Hermione_|_6_|_00000000000000000000000000000002
	// African cucumber_|_Ursula_|_6_|_00000000000000000000000000000003
}

func ExampleProcRunner_subprocessError() {
	runner, _ := NewProcRunner(&Parameters{
		Path: cli2.TestCliPath,
		Args: []string{
			"--" + cli2.FlagDisablePrompt,
			"--" + cli2.FlagRowToErrorOn, "4",
		},
		ExitCommand: cli2.CmdQuit,
		OutSentinel: cli2.MakeOutSentinelCommander(),
	})
	commander := cmdrs.NewPrintingCommander("query limit 3", os.Stdout)
	// Yields three lines.
	assertNoErr(runner.RunIt(commander, testingTimeout))

	// Query again, but ask for a row beyond the row that triggers a DB error.
	// Because of the nature of output streams, there's no way to know
	// when the error will show up in the combined output.  It might come
	// out first, last, or anywhere in the middle relative to lines from stdOut,
	// so this test must not be fragile to the order.
	commander.Reset()
	commander.Command = "query limit 7"
	// This will yield three "good lines", and one error line.
	assertNoErr(runner.RunIt(commander, testingTimeout))

	commander.Reset()
	commander.Command = "query limit 2"
	// Yields two lines.
	assertNoErr(runner.RunIt(commander, testingTimeout))

	// There should be nine (3 + 3 + 1 + 2) lines in the output.

	// Unordered output:
	// Cempedak_|_Bamberga_|_4_|_00000000000000000000000000000001
	// Buddha's hand_|_Hermione_|_6_|_00000000000000000000000000000002
	// African cucumber_|_Ursula_|_6_|_00000000000000000000000000000003
	// error! touching row 4 triggers this error
	// Currant_|_Alauda_|_5_|_00000000000000000000000000000001
	// Banana_|_Egeria_|_5_|_00000000000000000000000000000002
	// Bilberry_|_Interamnia_|_2_|_00000000000000000000000000000003
	// Cherimoya_|_Palma_|_6_|_00000000000000000000000000000001
	// Abiu_|_Metis_|_3_|_00000000000000000000000000000002
}
