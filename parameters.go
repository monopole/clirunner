package clirunner

import (
	"fmt"

	"github.com/monopole/clirunner/ifc"
)

// Parameters is a bag of parameters for ProcRunner.
type Parameters struct {
	// WorkingDir is the working directory of the CLI process.
	WorkingDir string

	// Path is the absolute or WorkingDir-relative path to the CLI's executable.
	Path string

	// Args has the arguments, flags and flag arguments for the CLI invocation.
	Args []string

	// ErrPrefix is added to the lines coming out of stdErr before combining
	// them with lines from stdOut.  Can be empty.  This is just a way
	// to help a Commander implementation more easily distinguish stdErr
	// from stdOut.
	// Example: "Err: "
	ErrPrefix string

	// ExitCommand is the command to send to gracefully exit the CLI.
	// If empty it won't be sent.  Regardless, the final thing sent to the
	// CLI subprocess will be an EOF on its stdIn.
	// Example: "quit"
	ExitCommand string

	// OutSentinel holds the command sent to the CLI after every command other
	// than the ExitCommand.  The OutSentinel knows how to scan output for a
	// particular sentinel value.
	//
	// If the command string is empty, it's presumed that the ProcRunner will rely
	// on the CLI to send a unique prompt to stdout, and the commander
	// will parse output looking for that prompt.
	//
	// Even when prompts are available, if they are short and not unambiguously
	// distinguishable from all possible command output, it's best to identify a
	// sentinel command to run instead.
	//
	//   Example: "echo pink elephants dance;"
	//   Look for: "pink elephants dance"
	//
	//   Example: "version;"
	//   Look for: "v1.2.3"
	//
	// The sentinel can be custom, but it's simplest to use an instance
	// of SimpleSentinelCommander, which can accommodate prompt detection.
	OutSentinel ifc.Commander

	// ErrSentinel is a command that intentionally triggers output on stderr,
	// e.g. a misspelled command, a command with a non-existent flag - something
	// that doesn't cause any real trouble.  In non nil, this is issued after
	// issuing command N, either before or after issuing the OutSentinel command.
	// ErrSentinel is used to be sure that any errors generated in the course of
	// running command N are swept up and accounted for before looking for errors
	// from command N+1.
	ErrSentinel ifc.Commander

	// CommandTerminator, if not 0, is appended to the end of every command.
	// This is merely a convenience for CLI's like mysql that want such things.
	//
	// Example: ';'
	CommandTerminator byte
}

// Validate looks for trouble and sets defaults.
func (p *Parameters) Validate() error {
	if p.Path == "" {
		return fmt.Errorf("must specify a Path")
	}
	if p.OutSentinel == nil {
		return fmt.Errorf("must specify OutSentinel")
	}
	// TODO: assure Path actually exists and
	// TODO: assure working dir actually exists.
	return nil
}
