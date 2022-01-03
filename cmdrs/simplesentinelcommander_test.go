package cmdrs_test

import (
	"strings"
	"testing"

	. "github.com/monopole/clirunner/cmdrs"
	"github.com/stretchr/testify/assert"
)

//goland:noinspection ALL
func TestSimpleSentinelCommander(t *testing.T) {
	var testCases = map[string]struct {
		input           []string
		expectedMatch   string
		expectedSuccess bool
	}{
		"empty": {
			expectedMatch: "",
		},
		"beginningOfLine": {
			input: strings.Split(`
Lorem ipsum dolor sit amet, consectetur adipiscing
elit. Sed nec congue ante. Cras eget urna mattis
nulla semper bibendum. Mauris mollis sollicitudin
PROMPT>pulvinar. Etiam nibh libero, iaculis a ante id,
rhoncus convallis felis. Donec molestie massa vitae
nisi convallis, ac finibus ipsum eleifend. 
`[1:], "\n"),
			expectedMatch:   `PROMPT>pulvinar. Etiam nibh libero, iaculis a ante id,`,
			expectedSuccess: true,
		},
		"midLine": {
			input: strings.Split(`
Lorem ipsum dolor sit amet, consectetur adipiscing
elit. Sed nec congue ante. Cras eget urna mattis
nulla semper bibendum. PROMPT>Mauris mollis sollicitudin
pulvinar. Etiam nibh libero, iaculis a ante id,
rhoncus convallis felis. Donec molestie massa vitae
nisi convallis, ac finibus ipsum eleifend. 
`[1:], "\n"),
			expectedMatch:   `nulla semper bibendum. PROMPT>Mauris mollis sollicitudin`,
			expectedSuccess: true,
		},
		"nope": {
			input: strings.Split(`
Lorem ipsum dolor sit amet, consectetur adipiscing
elit. Sed nec congue ante. Cras eget urna mattis
nulla semper bibendum. Mauris mollis sollicitudin
pulvinar. Etiam nibh libero, iaculis a ante id,
rhoncus convallis felis. Donec molestie massa vitae
nisi convallis, ac finibus ipsum eleifend. 
`[1:], "\n"),
			expectedMatch:   ``,
			expectedSuccess: false,
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			ssc := &SimpleSentinelCommander{
				Command: "not used here",
				Value:   `PROMPT>`,
			}
			assert.False(t, ssc.Success())
			for i := range tc.input {
				assert.NoError(t, WriteString(ssc, tc.input[i]))
			}
			if tc.expectedSuccess {
				assert.True(t, ssc.Success())
				assert.Equal(t, tc.expectedMatch, ssc.Match())
			} else {
				assert.False(t, ssc.Success())
			}
			ssc.Reset()
			assert.False(t, ssc.Success())
		})
	}
}
