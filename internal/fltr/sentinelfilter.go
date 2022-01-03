package fltr

import (
	"bytes"
	"fmt"
	"github.com/monopole/clirunner/ifc"
	"io"
	"sync"
	"time"
)

const (
	// defaultSentinelDuration is short for a human, but long enough for simple,
	// quick commands (the kind one wants as a sentinel) to finish.
	defaultSentinelDuration = 3 * time.Second
	// LineFeed makes it easier to find places where a linefeed is used.
	LineFeed = '\n'
)

// SentinelFilter is used by ProcRunner to orchestrate command lifetimes.
// It allows a command to be issued to some stdIn writer, assuring that the
// command line is properly terminated.  After issuing the command, the
// SentinelFilter optionally follows up by issuing zero, one or two sentinel
// commands, and watching for predefined sentinel value output from these
// commands to appear on stdOut and/or stdErr.  When these sentinel values are
// seen, one knows that theCmdr must be done.
type SentinelFilter struct {
	stdIn       io.Writer     // presumably the stdIn of some process.
	theCmdr     ifc.Commander // the command we're running
	cmdrLock    sync.Mutex    // lock on theCmdr to coordinate writing
	outSentinel ifc.Commander // for stdOut (required; command can be empty)
	errSentinel ifc.Commander // for stdErr (optional but recommended)
	terminator  byte          // command line terminator (a convenience)
	running     bool          // true if a command is running.
}

// MakeSentinelFilter returns an instance of SentinelFilter.
func MakeSentinelFilter(
	os ifc.Commander, es ifc.Commander, t byte) *SentinelFilter {
	if os == nil {
		panic("must have an outSentinel")
	}
	if es != nil && os.String() == es.String() {
		panic("the out and err sentinel commands must differ")
		// The success criterion - the things being looked for - should also differ.
	}
	return &SentinelFilter{outSentinel: os, errSentinel: es, terminator: t}
}

// BeginRun writes the command string to the given writer, presumably
// the stdIn of some process, to start a command run.  It doesn't block.
// It assures the command string is properly terminated.
// It returns the actual command sent (possibly with different termination),
// and any writer error.
func (cw *SentinelFilter) BeginRun(c ifc.Commander, w io.Writer) (string, error) {
	cw.stdIn = w
	cw.theCmdr = c
	return cw.issueCommand(c.String())
}

func (cw *SentinelFilter) issueCommand(c string) (string, error) {
	if len(c) == 0 {
		return "", nil
	}
	fullCmd := AssureCmdLineTermination([]byte(c), cw.terminator)
	n, err := io.WriteString(cw.stdIn, fullCmd)
	if err != nil || n != len(fullCmd) {
		err = fmt.Errorf(
			"wrote %d of %d bytes of command %q - %w", n, len(fullCmd), fullCmd, err)
	}
	// Can call BeginRun even while running, otherwise we couldn't send sentinel
	// commands to follow a 'normal' command.
	cw.running = true
	return fullCmd, err
}

func (cw *SentinelFilter) resetFilter() {
	cw.running = false
	cw.outSentinel.Reset()
	if cw.errSentinel != nil {
		cw.errSentinel.Reset()
	}
}

// IsRunning returns true if we've called BeginRun but not yet seen a sentinel
// to indicate a completion.
func (cw *SentinelFilter) IsRunning() bool {
	return cw.running
}

// IssueSentinelsAndFilter defines command completion.
//
// The method optionally issues zero, one or two sentinel commands to assure the
// appearance of a sentinel value on chOut and/or chErr.  If no sentinel
// commands are issued, it's assumed that the subprocess will send a prompt to
// stdOut or stdErr that can be recognized as a sentinel value.
//
// This method reads lines from chOut and chErr. It doesn't return until it sees
// the requested sentinel values (one or two), or until the given duration
// passes.
//
// If a line from chOut or chErr doesn't contain a sentinel, it's passed to the
// Commander for processing.  The Commander always sees stdOut and stdErr.
//
// A big assumption here is that stdIn is connected to some process input, and
// that chOut and chErr respectively represent the stdOut and stdErr of that
// same process (as would be arranged by an instance of ProcRunner).
func (cw *SentinelFilter) IssueSentinelsAndFilter(
	chOut <-chan []byte, // scan this for command output
	chErr <-chan []byte, // scan this for command errors
	d time.Duration, // time limit on finding the sentinel value
) (err error) {
	if !cw.IsRunning() {
		return fmt.Errorf("nothing is running")
	}
	defer cw.resetFilter()
	if d == 0 {
		d = defaultSentinelDuration
	}
	// If this is empty, the client is presumably depending on the CLI to send
	// a prompt, and the outSentinel knows how to recognize the prompt.
	if _, err = cw.issueCommand(cw.outSentinel.String()); err != nil {
		return
	}
	// Send the error sentinel command (if non-empty).  This should be a command
	// that does nothing more than generate some harmless error message on stdErr,
	// e.g. an attempt to use a non-existent command.
	if cw.errSentinel != nil {
		if _, err = cw.issueCommand(cw.errSentinel.String()); err != nil {
			return
		}
	}

	done := make(chan error)
	go cw.filterForSentinels(done, chOut, chErr)
	select {
	case <-time.After(d):
		err = cw.expirationError(d)
	case err = <-done:
	}
	return
}

// filterForSentinels returns after sentinel success on both stdOut and stdErr.
func (cw *SentinelFilter) filterForSentinels(
	done chan<- error, chOut <-chan []byte, chErr <-chan []byte,
) {
	defer close(done)
	var errOut, errErr error
	var scanWg sync.WaitGroup
	scanWg.Add(1)
	go cw.filterForSentinel("Out", &errOut, &scanWg, cw.outSentinel, chOut)
	if cw.errSentinel != nil {
		scanWg.Add(1)
		go cw.filterForSentinel("Err", &errErr, &scanWg, cw.errSentinel, chErr)
	} else {
		go cw.passThru("Err", &errErr, chErr)
	}
	scanWg.Wait()
	if errOut != nil {
		done <- errOut
		return
	}
	if errErr != nil {
		done <- errErr
	}
}

func (cw *SentinelFilter) filterForSentinel(
	title string, err *error,
	wg *sync.WaitGroup, sentinel ifc.Commander, ch <-chan []byte) {
	defer wg.Done()
	for {
		line, stillOpen := <-ch
		if !stillOpen {
			*err = fmt.Errorf(
				"std%s closed while or before running %q, no sentinel detected",
				title, cw.theCmdr.String())
			return
		}
		panicIfNotActuallyALine(line)
		if !sentinel.Success() {
			// Send the line to the sentinel value detector first,
			// to see if we're done.
			if _, *err = sentinel.Write(line); *err != nil {
				// Catastrophe of some kind.
				return
			}
		}
		if sentinel.Success() {
			// The line has the sentinel value; we're done.
			return
		}
		// Pass the line to the current commander for processing.
		cw.cmdrLock.Lock()
		// There are two threads that might write this.
		if _, *err = cw.theCmdr.Write(line); *err != nil {
			// Catastrophe of some kind.
			cw.cmdrLock.Unlock()
			return
		}
		cw.cmdrLock.Unlock()
	}
}

func (cw *SentinelFilter) passThru(
	title string, err *error, ch <-chan []byte) {
	for {
		line, stillOpen := <-ch
		if !stillOpen {
			return
		}
		panicIfNotActuallyALine(line)
		// Pass the line to the current commander for processing.
		cw.cmdrLock.Lock()
		// There are two threads that might write this.
		if _, *err = cw.theCmdr.Write(line); *err != nil {
			// Catastrophe of some kind.
			cw.cmdrLock.Unlock()
			return
		}
		cw.cmdrLock.Unlock()
	}
}

// Paranoia check; make sure all lines coming back are indeed "lines"
// in the sense that they do not contain a linefeed.
func panicIfNotActuallyALine(line []byte) {
	if bytes.Contains(line, []byte("\n")) {
		// This means parsing has failed dramatically, likely because
		// of a mistake in writing a shared buffer.  No point in continuing.
		panic(fmt.Errorf(
			"line %q should not have a linefeed in it", string(line)))
	}
}

func (cw *SentinelFilter) expirationError(d time.Duration) error {
	c := cw.theCmdr.String()
	msg := fmt.Sprintf(
		"in command %q, time %s expired before detection of ", c, d)
	if cw.outSentinel.String() == "" {
		return fmt.Errorf(msg + "prompt")
	}
	return fmt.Errorf(
		msg+"output from sentinel command %q", cw.outSentinel.String())
}

// AssureCmdLineTermination assures that the last characters of a command line
// are correct.
func AssureCmdLineTermination(c []byte, terminator byte) string {
	if c[len(c)-1] == LineFeed {
		// Slice it off avoid confusion, replace momentarily.  Cap() unchanged.
		c = c[:len(c)-1]
	}
	if terminator > 0 && c[len(c)-1] != terminator {
		c = append(c, terminator)
	}
	return string(append(c, LineFeed))
}
