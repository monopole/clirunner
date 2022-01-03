package fltr_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/monopole/clirunner/internal/testcli/tstcli"

	"github.com/monopole/clirunner/cmdrs"
	. "github.com/monopole/clirunner/internal/fltr"
	. "github.com/monopole/clirunner/internal/testing"
	"github.com/stretchr/testify/assert"
)

func TestSentinelFilter_BeginRun(t *testing.T) {
	cw := MakeSentinelFilter(tstcli.MakeOutSentinelCommander(), nil, ';')
	assert.False(t, cw.IsRunning())
	cmdr := &cmdrs.KondoCommander{Command: "kondo"}
	var stdIn bytes.Buffer
	c, err := cw.BeginRun(cmdr, &stdIn)
	assert.NoError(t, err)
	assert.Equal(t, "kondo;\n", c)
	assert.Equal(t, "kondo;\n", stdIn.String())
	assert.True(t, cw.IsRunning())
}

func TestSentinelFilter_WatchAndWait_timeout(t *testing.T) {
	sentinel := tstcli.MakeOutSentinelCommander()
	cmdr := cmdrs.NewHoardingCommander("hoard")
	cw := MakeSentinelFilter(sentinel, nil, ';')
	var stdIn bytes.Buffer
	_, err := cw.BeginRun(cmdr, &stdIn)
	assert.NoError(t, err)
	assert.Equal(t, "hoard;\n", stdIn.String())
	stdOut := make(chan []byte)
	err = cw.IssueSentinelsAndFilter(stdOut, nil, 1*time.Second)
	if !assert.Error(t, err) {
		t.Fatalf("expected timeout")
	}
	assert.Contains(
		t, err.Error(),
		"time 1s expired before detection of output from sentinel command")
}

func TestSentinelFilter_WatchAndWait_noTimeout(t *testing.T) {
	sentinel := tstcli.MakeOutSentinelCommander()
	cmdr := cmdrs.NewHoardingCommander("hoard")
	cw := MakeSentinelFilter(sentinel, nil, ';')
	var stdIn bytes.Buffer
	_, err := cw.BeginRun(cmdr, &stdIn)
	assert.NoError(t, err)
	assert.Equal(t, "hoard;\n", stdIn.String())
	stdOut := make(chan []byte)
	go func() {
		stdOut <- []byte("these lines represent output")
		stdOut <- []byte("from command n")
		stdOut <- []byte(sentinel.Value)
		// Anything after the sentinel value should not be captured by
		// our hoarding commander; subsequent lines simulate output from
		// the next command.
		stdOut <- []byte("and these lines represent output")
		stdOut <- []byte("from command n+1")
	}()
	stdErr := make(chan []byte)
	// write nothing to stdErr
	assert.NoError(t, cw.IssueSentinelsAndFilter(stdOut, stdErr, 1*time.Second))
	assert.Equal(t, "hoard;\n"+sentinel.Command+";\n", stdIn.String())
	assert.Equal(t, `
these lines represent output
from command n
`[1:], cmdr.Result())
}

func TestSentinelFilter_WatchAndWait_bothOutAndError(t *testing.T) {
	outSentinel := tstcli.MakeOutSentinelCommander()
	errSentinel := tstcli.MakeErrSentinelCommander()
	cmdr := cmdrs.NewHoardingCommander("hoard")
	cw := MakeSentinelFilter(outSentinel, errSentinel, ';')
	var stdIn bytes.Buffer
	_, err := cw.BeginRun(cmdr, &stdIn)
	assert.NoError(t, err)
	assert.Equal(t, "hoard;\n", stdIn.String())
	stdOut := make(chan []byte)
	go func() {
		stdOut <- []byte("these lines represent output")
		stdOut <- []byte("from command n")
		stdOut <- []byte(outSentinel.Value)
		// Anything after the outSentinel value should not be captured by
		// our hoarding commander; subsequent lines simulate output from
		// the next command.
		stdOut <- []byte("and these lines represent output")
		stdOut <- []byte("from command n+1")
	}()
	stdErr := make(chan []byte)
	go func() {
		stdErr <- []byte("oh no some error from command n!")
		stdErr <- []byte(errSentinel.Value)
		// Anything after the outSentinel value should not be captured by
		// our hoarding commander; subsequent lines simulate output from
		// the next command.
		stdErr <- []byte("and this line is an error from command n+1")
	}()
	assert.NoError(t, cw.IssueSentinelsAndFilter(stdOut, stdErr, 1*time.Second))
	assert.Equal(t, `hoard;
`+outSentinel.Command+`;
`+errSentinel.Command+`;
`, stdIn.String())
	AssertEqualAnyOrder(t, `
these lines represent output
from command n
oh no some error from command n!
`[1:], cmdr.Result())
}

func TestSentinelFilter_WatchAndWait_diesBeforeSentinel(t *testing.T) {
	outSentinel := tstcli.MakeOutSentinelCommander()
	errSentinel := tstcli.MakeErrSentinelCommander()
	cmdr := cmdrs.NewHoardingCommander("hoard")
	cw := MakeSentinelFilter(outSentinel, errSentinel, ';')
	var stdIn bytes.Buffer
	_, err := cw.BeginRun(cmdr, &stdIn)
	assert.NoError(t, err)
	assert.Equal(t, "hoard;\n", stdIn.String())
	stdOut := make(chan []byte)
	go func() {
		stdOut <- []byte("these lines represent output")
		stdOut <- []byte("from command n")
		stdOut <- []byte(outSentinel.Value)
		close(stdOut)
	}()
	stdErr := make(chan []byte)
	go func() {
		stdErr <- []byte("oh no some error from command n!")
		// Don't send the error sentinel value.
		// stdOut <- []byte(errSentinel.Value)
		close(stdErr)
	}()
	err = cw.IssueSentinelsAndFilter(stdOut, stdErr, 1*time.Second)
	if !assert.Error(t, err) {
		t.Fatalf("expected error")
	}
	assert.Contains(
		t, err.Error(),
		`stdErr closed while or before running "hoard", no sentinel detected`)
	assert.Equal(t, `hoard;
`+outSentinel.Command+`;
`+errSentinel.Command+`;
`, stdIn.String())
	AssertEqualAnyOrder(t, `
these lines represent output
from command n
oh no some error from command n!
`[1:], cmdr.Result())
}

// Make sure the end of the command is as expected.
func TestAssureCmdLineTermination(t *testing.T) {
	var empty byte

	testCases := map[string]struct {
		line     string
		term     byte
		expected string
	}{
		"t1": {
			line:     "hey;\n",
			term:     ';',
			expected: "hey;\n",
		},
		"t2": {
			line:     "hey\n",
			term:     ';',
			expected: "hey;\n",
		},
		"t3": {
			line:     "hey\n",
			term:     ';',
			expected: "hey;\n",
		},
		"t4": {
			line:     "hey;\n",
			term:     empty,
			expected: "hey;\n",
		},
		"t5": {
			line:     "hey\n",
			term:     empty,
			expected: "hey\n",
		},
		"t6": {
			line:     "hey",
			term:     empty,
			expected: "hey\n",
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			assert.Equal(
				t, tc.expected, AssureCmdLineTermination([]byte(tc.line), tc.term))
		})
	}
}
