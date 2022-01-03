package cmdrs

import (
	"github.com/monopole/clirunner/ifc"
)

// KondoCommander quietly discards everything sent to Write
// and always reports Success true.
// Use this when you just want to run a command and don't care
// about the command's output.
type KondoCommander struct {
	Command string
}

var _ ifc.Commander = &KondoCommander{}

// Write accepts input to discard.
func (c *KondoCommander) Write(_ []byte) (int, error) { return 0, nil }

// Success always returns true.
func (c *KondoCommander) Success() bool { return true }

// Reset does nothing.
func (c *KondoCommander) Reset() {}

// String returns the command string.
func (c *KondoCommander) String() string { return c.Command }
