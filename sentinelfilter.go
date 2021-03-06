package clirunner

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"time"
)

const (
	// defaultSentinelDuration is short for a human, but long enough for simple,
	// quick commands (the kind one wants as a sentinel) to finish.
	defaultSentinelDuration = 3 * time.Second
	// lineFeed makes it easier to find places where a linefeed is used.
	lineFeed = '\n'
)

// sentinelFilter is used by ProcRunner to orchestrate command lifetimes.
// It allows a command to be issued to some stdIn writer, assuring that the
// command line is properly terminated.  After issuing the command, the
// sentinelFilter optionally follows up by issuing zero, one or two sentinel
// commands, and watching for predefined sentinel value output from these
// commands to appear on stdOut and/or stdErr.  When these sentinel values are
// seen, one knows that theCmdr must be done.
type sentinelFilter struct {
	stdIn       io.Writer  // presumably the stdIn of some process.
	theCmdr     Commander  // the command we're running
	cmdrLock    sync.Mutex // lock on theCmdr to coordinate writing
	outSentinel Commander  // for stdOut (required; command can be empty)
	errSentinel Commander  // for stdErr (optional but recommended)
	terminator  byte       // command line terminator (a convenience)
	running     bool       // true if a command is running.
}

// makeSentinelFilter returns an instance of sentinelFilter.
func makeSentinelFilter(
	os Commander, es Commander, t byte) *sentinelFilter {
	if os == nil {
		panic("must have an outSentinel")
	}
	if es != nil && os.String() == es.String() {
		panic("the out and err sentinel commands must differ")
		// The success criterion - the things being looked for - should also differ.
	}
	return &sentinelFilter{outSentinel: os, errSentinel: es, terminator: t}
}

// BeginRun writes the command string to the given writer, presumably
// the stdIn of some process, to start a command run.  It doesn't block.
// It assures the command string is properly terminated.
// It returns the actual command sent (possibly with different termination),
// and any writer error.
func (cw *sentinelFilter) BeginRun(c Commander, w io.Writer) (string, error) {
	cw.stdIn = w
	cw.theCmdr = c
	return cw.issueCommand(c.String())
}

func (cw *sentinelFilter) issueCommand(c string) (string, error) {
	if len(c) == 0 {
		return "", nil
	}
	logger.Printf("issueCommand called with: %q\n", c)
	fullCmd := assureCmdLineTermination([]byte(c), cw.terminator)
	n, err := io.WriteString(cw.stdIn, fullCmd)
	logger.Printf("wrote command to subprocess stdIn: %q\n", fullCmd)

	if err != nil || n != len(fullCmd) {
		err = fmt.Errorf(
			"wrote %d of %d bytes of command %q - %w", n, len(fullCmd), fullCmd, err)
	}
	// Can call BeginRun even while running, otherwise we couldn't send sentinel
	// commands to follow a 'normal' command.
	cw.running = true
	return fullCmd, err
}

func (cw *sentinelFilter) resetFilter() {
	cw.running = false
	cw.outSentinel.Reset()
	if cw.errSentinel != nil {
		cw.errSentinel.Reset()
	}
}

// isRunning returns true if we've called BeginRun but not yet seen a sentinel
// to indicate a completion.
func (cw *sentinelFilter) isRunning() bool {
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
func (cw *sentinelFilter) IssueSentinelsAndFilter(
	chOut <-chan []byte, // scan this for command output
	chErr <-chan []byte, // scan this for command errors
	timeOut time.Duration, // time limit on finding the sentinel value
) (err error) {
	if !cw.isRunning() {
		return fmt.Errorf("nothing is running")
	}
	defer cw.resetFilter()
	if timeOut == 0 {
		timeOut = defaultSentinelDuration
	}
	logger.Printf("entering IssueSentinelsAndFilter with timeOut = %s", timeOut)
	logger.Printf("out sentinel = %q", cw.outSentinel.String())

	// If this is empty, the client is presumably depending on the CLI to send
	// a prompt, and the outSentinel knows how to recognize the prompt.
	if _, err = cw.issueCommand(cw.outSentinel.String()); err != nil {
		return
	}
	if err != nil {
		logger.Printf("issueCommand err = %s", err.Error())
		return err
	}

	// Send the error sentinel command (if non-empty).  This should be a command
	// that does nothing more than generate some harmless error message on stdErr,
	// e.g. an attempt to use a non-existent command.
	if cw.errSentinel != nil {
		logger.Printf("err sentinel = %v", cw.errSentinel.String())
		if _, err = cw.issueCommand(cw.errSentinel.String()); err != nil {
			return
		}
	}

	done := make(chan error)
	go cw.filterForSentinels(done, chOut, chErr)

	logger.Printf("Waiting %s to see sentinel\n", timeOut)

	select {
	case <-time.After(timeOut):
		err = cw.expirationError(timeOut)
	case err = <-done: // This is the one we want, hopefully with err==nil
	}
	return
}

// filterForSentinels returns after sentinel success on both stdOut and stdErr.
func (cw *sentinelFilter) filterForSentinels(
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
		logger.Println("filterForSentinels found errOut = " + errOut.Error())
		done <- errOut
		return
	}
	if errErr != nil {
		logger.Println("filterForSentinels found errErr = " + errOut.Error())
		done <- errErr
	}
}

func (cw *sentinelFilter) filterForSentinel(
	title string, err *error,
	wg *sync.WaitGroup, sentinel Commander, ch <-chan []byte) {
	defer wg.Done()
	logger.Printf("starting %q filter for command %q", title, sentinel)
	for {
		line, stillOpen := <-ch
		logger.Printf("outCh returns line: %s", string(line))
		if !stillOpen {
			logger.Println("outCh appears closed")
			*err = fmt.Errorf(
				"std%s closed while or before running %q, no sentinel detected",
				title, cw.theCmdr.String())
			return
		}
		panicIfNotActuallyALine(line)
		if !sentinel.Success() {
			logger.Printf("sending line %q to sentinel\n", string(line))
			// Send the line to the sentinel value detector first,
			// to see if we're done.
			if _, *err = sentinel.Write(line); *err != nil {
				logger.Printf("Catastrophe err=%s\n", *err)
				// Catastrophe of some kind.
				return
			}
		}
		if sentinel.Success() {
			logger.Printf("sentinel success!\n")
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

func (cw *sentinelFilter) passThru(
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

func (cw *sentinelFilter) expirationError(d time.Duration) error {
	c := cw.theCmdr.String()
	msg := fmt.Sprintf(
		"in command %q, time %s expired before detection of ", c, d)
	if cw.outSentinel.String() == "" {
		return fmt.Errorf(msg + "prompt")
	}
	return fmt.Errorf(
		msg+"output from sentinel command %q", cw.outSentinel.String())
}

// assureCmdLineTermination assures that the last characters of a command line
// are correct.
func assureCmdLineTermination(c []byte, terminator byte) string {
	if c[len(c)-1] == lineFeed {
		// Slice it off avoid confusion, replace momentarily.  Cap() unchanged.
		c = c[:len(c)-1]
	}
	if terminator > 0 && c[len(c)-1] != terminator {
		c = append(c, terminator)
	}
	return string(append(c, lineFeed))
}
