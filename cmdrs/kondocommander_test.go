package cmdrs_test

import (
	"testing"

	. "github.com/monopole/clirunner/cmdrs"
	"github.com/stretchr/testify/assert"
)

func TestKondoCommander(t *testing.T) {
	var testCases = map[string]struct {
		input   []string
		command string
	}{
		"t1": {
			input:   []string{"hello", "there"},
			command: "help",
		},
		"noInput": {
			command: "whatever",
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			c := &KondoCommander{Command: tc.command}
			assert.Equal(t, tc.command, c.String())
			assert.True(t, c.Success())
			for i := range tc.input {
				assert.NoError(t, WriteString(c, tc.input[i]))
			}
			assert.True(t, c.Success())
			c.Reset()
			assert.True(t, c.Success())
		})
	}
}
