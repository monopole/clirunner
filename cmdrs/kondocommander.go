package cmdrs

// KondoCommander quietly discards everything sent to Write
// and always reports Success true.
// Use this when you just want to run a command and don't care
// about the command's output.
type KondoCommander struct {
	Command string
}

// Write accepts input to discard.
// Great place to debugging output.
func (c *KondoCommander) Write(s []byte) (int, error) {
	// For debugging: fmt.Printf("Kondo saw: %q\n", string(s))
	return 0, nil
}

// Success always returns true.
func (c *KondoCommander) Success() bool { return true }

// Reset does nothing.
func (c *KondoCommander) Reset() {}

// String returns the command string.
func (c *KondoCommander) String() string { return c.Command }
