package cmdrs

import (
	"bytes"
)

// SimpleSentinelCommander is a Commander that asserts Success if it sees
// Value anywhere in the output of Command.
type SimpleSentinelCommander struct {
	Command string // the command, e.g. "echo Rumplestilskin"
	Value   string // the sentinel value to look for, e.g. "Rumplestilskin".
	success bool   // internal state
	// match stores the entire winning line that contains Value.
	// Handy for debugging.
	match string
}

func (c *SimpleSentinelCommander) String() string { return c.Command }

// Write looks for Value anywhere in the line (so it had better be unambiguous).
func (c *SimpleSentinelCommander) Write(b []byte) (int, error) {
	if bytes.Contains(b, []byte(c.Value)) {
		c.match = string(b)
		c.success = true
	}
	return 0, nil
}

// Reset resets everything.
func (c *SimpleSentinelCommander) Reset() {
	c.match = ""
	c.success = false
}

// Success returns true if Value found.
func (c *SimpleSentinelCommander) Success() bool { return c.success }

// Match returns the winning line.
func (c *SimpleSentinelCommander) Match() string { return c.match }
