package cmdrs

import (
	"bytes"
	"github.com/monopole/clirunner/ifc"
)

// HoardingCommander keeps everything sent into Write.
// Handy for tests, debugging etc.
type HoardingCommander struct {
	data bytes.Buffer
	KondoCommander
}

var _ ifc.Commander = &HoardingCommander{}

// NewHoardingCommander returns a new instance of HoardingCommander.
func NewHoardingCommander(c string) *HoardingCommander {
	return &HoardingCommander{
		KondoCommander: KondoCommander{Command: c},
	}
}

// Write accepts input to store in a buffer.
func (c *HoardingCommander) Write(b []byte) (int, error) {
	_, err := c.data.Write(b)
	if err != nil {
		return 0, err
	}
	// Restore the LineFeed that was stripped by the text scanner.
	return 0, c.data.WriteByte('\n')
}

// Reset clears the internal buffer.
func (c *HoardingCommander) Reset() { c.data.Reset() }

// Result returns the buffer contents as a string.
func (c *HoardingCommander) Result() string { return c.data.String() }
