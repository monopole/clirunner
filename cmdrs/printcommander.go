package cmdrs

import (
	"fmt"
	"github.com/monopole/clirunner/ifc"
	"io"
)

var _ ifc.Commander = &PrintingCommander{}

// PrintingCommander echos everything to stdout.
// Only useful for examples, tests, debugging, etc.
type PrintingCommander struct {
	out io.Writer
	KondoCommander
}

// NewPrintingCommander returns a new instance of PrintingCommander.
func NewPrintingCommander(c string, o io.Writer) *PrintingCommander {
	return &PrintingCommander{
		out:            o,
		KondoCommander: KondoCommander{Command: c},
	}
}

// Write accepts input to print.
func (c *PrintingCommander) Write(b []byte) (int, error) {
	return fmt.Fprintln(c.out, string(b))
}
