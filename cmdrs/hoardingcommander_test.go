package cmdrs_test

import (
	"testing"

	. "github.com/monopole/clirunner/cmdrs"
	"github.com/stretchr/testify/assert"
)

func TestHoardingCommander(t *testing.T) {
	var testCases = map[string]struct {
		command  string
		input    []string
		expected string
	}{
		"t1": {
			command: "help",
			input:   []string{"hello", "there"},
			expected: `
hello
there
`[1:],
		},
		"noInput": {
			command:  "whatever",
			expected: ``,
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			c := NewHoardingCommander(tc.command)
			assert.Equal(t, tc.command, c.String())
			assert.True(t, c.Success())
			for i := range tc.input {
				assert.NoError(t, WriteString(c, tc.input[i]))
			}
			assert.Equal(t, tc.expected, c.Result())
			assert.True(t, c.Success())
			c.Reset()
			assert.Equal(t, "", c.Result())
		})
	}
}
