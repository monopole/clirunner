package clirunner_test

import (
	"testing"
	"time"

	. "github.com/monopole/clirunner"
	. "github.com/monopole/clirunner/cmdrs"
	"github.com/monopole/clirunner/internal/testcli/tstcli"
	. "github.com/monopole/clirunner/internal/testing"
	"github.com/stretchr/testify/assert"
)

const (
	nonexistentCommandPath = "beamMeUpScotty"
	testingErrPrefix       = "yikes: "
	testingTimeout         = 5 * time.Second
)

func assertNoErr(err error) {
	if err != nil {
		panic(err)
	}
}

func TestNewRunner(t *testing.T) {
	r, err := NewProcRunner(&Parameters{
		Path:        nonexistentCommandPath,
		OutSentinel: tstcli.MakeOutSentinelCommander(),
	})
	assert.NoError(t, err)
	assert.NotNil(t, r)
}

func TestRunner_Run_BadPath(t *testing.T) {
	r, err := NewProcRunner(&Parameters{
		Path:        nonexistentCommandPath,
		OutSentinel: tstcli.MakeOutSentinelCommander(),
	})
	assert.NoError(t, err)
	err = r.RunIt(NewHoardingCommander(tstcli.CmdQuery+" limit 1"), testingTimeout)
	if !assert.Error(t, err) {
		t.Fatal("expecting an error")
	}
	assert.Contains(t, err.Error(), "executable file not found")
}

func TestRunner_Run_NoSentinelCommander(t *testing.T) {
	_, err := NewProcRunner(&Parameters{
		Path: nonexistentCommandPath,
	})
	if !assert.Error(t, err) {
		t.Fatal("expecting an error")
	}
	assert.Contains(t, err.Error(), "must specify OutSentinel")
}

func TestRunner_Run_ForgotTheCommander(t *testing.T) {
	runner, err := NewProcRunner(&Parameters{
		Path:        tstcli.TestCliPath,
		OutSentinel: tstcli.MakeOutSentinelCommander(),
	})
	assert.NoError(t, err)
	err = runner.RunIt(nil, testingTimeout)
	if !assert.Error(t, err) {
		t.Fatal("expecting an error")
	}
	assert.Contains(t, err.Error(), "provide a Commander")
	assert.NoError(t, runner.Close())
}

// Using a prompt-only sentinel is not well tested.
func TestRunner_Run_YouForgotToDisableThePrompt(t *testing.T) {
	runner, err := NewProcRunner(&Parameters{
		Path: tstcli.TestCliPath,
		// intentionally leave prompt enabled.
		ExitCommand: tstcli.CmdQuit,
		OutSentinel: tstcli.MakeOutSentinelCommander(),
	})
	assert.NoError(t, err)
	commander := NewHoardingCommander(tstcli.CmdQuery + " limit 1")
	assert.NoError(t, runner.RunIt(commander, testingTimeout))
	assert.Equal(t, `
hey<1>Cempedak_|_Bamberga_|_4_|_00000000000000000000000000000001
`[1:], commander.Result())
	assert.NoError(t, runner.Close())
}

/*

Need a v2 here.  there's too much synchrony in the current impl.
RunIt needs a select loop, looking for completion of either
the command or the process itself.  currently there is no
loop at all, it's just stupid.

*/
func TestRunner_Run_FailOnStartup(t *testing.T) {
	runner, err := NewProcRunner(&Parameters{
		Path:        tstcli.TestCliPath,
		Args:        []string{"--" + tstcli.FlagFailOnStartup},
		ExitCommand: tstcli.CmdQuit,
		OutSentinel: tstcli.MakeOutSentinelCommander(),
	})
	assert.NoError(t, err)
	commander := NewHoardingCommander(tstcli.CmdQuery + " limit 5")
	assert.Error(t, runner.RunIt(commander, testingTimeout))
	assert.Error(t, runner.Close())
}

func TestRunner_Run_HappyQuery(t *testing.T) {
	runner, err := NewProcRunner(&Parameters{
		Path:        tstcli.TestCliPath,
		Args:        []string{"--" + tstcli.FlagDisablePrompt},
		ExitCommand: tstcli.CmdQuit,
		OutSentinel: tstcli.MakeOutSentinelCommander(),
	})
	assert.NoError(t, err)
	commander := NewHoardingCommander(tstcli.CmdQuery + " limit 5")
	assert.NoError(t, runner.RunIt(commander, testingTimeout))
	assert.Equal(t, `
Cempedak_|_Bamberga_|_4_|_00000000000000000000000000000001
Buddha's hand_|_Hermione_|_6_|_00000000000000000000000000000002
African cucumber_|_Ursula_|_6_|_00000000000000000000000000000003
Currant_|_Alauda_|_5_|_00000000000000000000000000000004
Banana_|_Egeria_|_5_|_00000000000000000000000000000005
`[1:], commander.Result())
	assert.NoError(t, runner.Close())
}

func TestRunner_Run_SentinelTimeoutOnLongRunningCommand(t *testing.T) {
	runner, err := NewProcRunner(&Parameters{
		Path:        tstcli.TestCliPath,
		Args:        []string{"--" + tstcli.FlagDisablePrompt},
		ExitCommand: tstcli.CmdQuit,
		OutSentinel: tstcli.MakeOutSentinelCommander(),
	})
	assert.NoError(t, err)
	// sleep exceeds SentinelDuration
	err = runner.RunIt(tstcli.MakeSleepCommander(4*time.Second), 1*time.Second)
	if !assert.Error(t, err) {
		t.Fatal("expecting an error")
	}
	assert.Contains(
		t, err.Error(), "time 1s expired before detection of output from sentinel")
}

func TestRunner_NoSentinelTimeoutOnShortRunningCommand(t *testing.T) {
	runner, err := NewProcRunner(&Parameters{
		Path:        tstcli.TestCliPath,
		Args:        []string{"--" + tstcli.FlagDisablePrompt},
		ExitCommand: tstcli.CmdQuit,
		OutSentinel: tstcli.MakeOutSentinelCommander(),
	})
	assert.NoError(t, err)
	assert.NoError(
		t, runner.RunIt(tstcli.MakeSleepCommander(1*time.Second), 4*time.Second))
	assert.NoError(t, runner.Close())
}

func TestRunner_ErrorInCommandNoErrorOnExit(t *testing.T) {
	runner, err := NewProcRunner(&Parameters{
		Path: tstcli.TestCliPath,
		Args: []string{
			"--" + tstcli.FlagDisablePrompt,
			"--" + tstcli.FlagRowToErrorOn, "4",
		},
		ExitCommand: tstcli.CmdQuit,
		OutSentinel: tstcli.MakeOutSentinelCommander(),
		ErrSentinel: tstcli.MakeErrSentinelCommander(),
	})
	assert.NoError(t, err)
	commander := NewHoardingCommander(tstcli.CmdQuery + " limit 3")
	assert.NoError(t, runner.RunIt(commander, testingTimeout))
	assert.Equal(t, `
Cempedak_|_Bamberga_|_4_|_00000000000000000000000000000001
Buddha's hand_|_Hermione_|_6_|_00000000000000000000000000000002
African cucumber_|_Ursula_|_6_|_00000000000000000000000000000003
`[1:], commander.Result())

	// Query again, but ask for a row beyond the row that triggers a DB error.
	commander.Reset()
	commander.Command = tstcli.CmdQuery + " limit 5"
	assert.NoError(t, runner.RunIt(commander, testingTimeout))
	assert.True(t, commander.Success())

	AssertEqualAnyOrder(t, `
Banana_|_Egeria_|_5_|_00000000000000000000000000000002
Bilberry_|_Interamnia_|_2_|_00000000000000000000000000000003
Currant_|_Alauda_|_5_|_00000000000000000000000000000001
error! touching row 4 triggers this error
`[1:], commander.Result())

	assert.NoError(t, runner.Close())
}

func TestRunner_ErrorInCommandOutputForcingExit(t *testing.T) {
	runner, err := NewProcRunner(&Parameters{
		Path: tstcli.TestCliPath,
		Args: []string{
			"--" + tstcli.FlagDisablePrompt,
			// Using this means any error will cause process exit.
			// So we cannot use an errSentinel, as it by definition causes an error.
			"--" + tstcli.FlagExitOnErr,
			"--" + tstcli.FlagRowToErrorOn, "4",
		},
		ExitCommand: tstcli.CmdQuit,
		OutSentinel: tstcli.MakeOutSentinelCommander(),
	})
	assert.NoError(t, err)
	commander := NewHoardingCommander(tstcli.CmdQuery + " limit 3")
	assert.NoError(t, runner.RunIt(commander, testingTimeout))
	assert.Equal(t, `
Cempedak_|_Bamberga_|_4_|_00000000000000000000000000000001
Buddha's hand_|_Hermione_|_6_|_00000000000000000000000000000002
African cucumber_|_Ursula_|_6_|_00000000000000000000000000000003
`[1:], commander.Result())
	assert.True(t, commander.Success())

	// Query again, but ask for a row beyond the row that triggers a DB error.
	// Since FlagExitOnErr is on, this causes the CLI to die.
	commander.Reset()
	commander.Command = tstcli.CmdQuery + " limit 5"
	err = runner.RunIt(commander, testingTimeout)
	if !assert.Error(t, err) {
		t.Fatal("expecting an error")
	}
	assert.Contains(t, err.Error(), "stdOut closed while or before")
	assert.Contains(t, err.Error(), "no sentinel detected")

	// This time we've captured the error from stdErr, because the process ended
	// and all the output was drained.
	AssertEqualAnyOrder(t, `
Currant_|_Alauda_|_5_|_00000000000000000000000000000001
Banana_|_Egeria_|_5_|_00000000000000000000000000000002
Bilberry_|_Interamnia_|_2_|_00000000000000000000000000000003
error! touching row 4 triggers this error
`[1:], commander.Result())

	err = runner.Close()
	if !assert.Error(t, err) {
		t.Fatal("expecting an error")
	}
	assert.Contains(t, err.Error(), "cannot close error state")
}

func TestRunner_ErrorPrefix(t *testing.T) {
	runner, err := NewProcRunner(&Parameters{
		Path: tstcli.TestCliPath,
		Args: []string{
			"--" + tstcli.FlagDisablePrompt,
			"--" + tstcli.FlagExitOnErr,
			"--" + tstcli.FlagRowToErrorOn, "4",
		},
		ErrPrefix:   testingErrPrefix,
		OutSentinel: tstcli.MakeOutSentinelCommander(),
	})
	assert.NoError(t, err)

	// Ask for a row beyond the row that triggers a DB error.
	// Since FlagExitOnErr is on, this causes the CLI to die.
	commander := NewHoardingCommander(tstcli.CmdQuery + " limit 5")
	err = runner.RunIt(commander, testingTimeout)
	if !assert.Error(t, err) {
		t.Fatal("expecting an error")
	}
	assert.Contains(t, err.Error(), "stdOut closed while or before running")
	assert.Contains(t, err.Error(), "no sentinel detected")

	AssertEqualAnyOrder(t, (`
Cempedak_|_Bamberga_|_4_|_00000000000000000000000000000001
Buddha's hand_|_Hermione_|_6_|_00000000000000000000000000000002
African cucumber_|_Ursula_|_6_|_00000000000000000000000000000003
` + testingErrPrefix + `error! touching row 4 triggers this error
`)[1:], commander.Result())
}
