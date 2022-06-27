package clirunner

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/monopole/clirunner/cmdrs"
	"github.com/pkg/errors"
)

// ProcRunner manages an interactive command line interpreter (CLI) subprocess.
// See nearby example and tests for usage.
//
// ProcRunner separates the problem of running a CLI from the problem of parsing
// the CLI's response to a particular command. The ProcRunner handles the
// former, and implementations of Commander handle the latter.  ProcRunner knows
// nothing about the commands in a given CLI. Its job is to start the CLI,
// accept instances of Commander, watch stdOut and stdErr, and run a series
// of Commander instances.
//
// So, one ProcRunner instance can be used to run any CLI (e.g. mysql, kubectl,
// mql, etc.), but the specific knowledge of a specific command and how to parse
// the output from that command must be expressed in an implementation of
// the Commander interface.
//
// For the ProcRunner to know when a Commander completes, it looks for a
// particular string called the "sentinel value" in the CLI's output stream
// (and optionally its error stream).  The sentinel value could simply be the
// value of the CLI prompt string. Or it could be the fixed, characteristic
// output of a particular "sentinel command", like "echo" or "version".  The
// sentinel value is analogous to the code word "Over" in a radio transmission.
//
// CLI's sometimes won't prompt to stdErr or stdOut if they detect that
// they are attached to a pipe on stdIn, so commands with a characteristic
// output are the only option for generating a sentinel.  Also, some prompts
// might not be unambiguously distinguishable in several thousand lines of data,
// so it's best to use a sentinel command rather than rely on a prompt to
// signal command completion.
//
// If the ProcRunner is prepared with a sentinel command, it will automatically
// issue the command inside the call to RunIt, immediately after issuing the
// command given to RunIt.  During the call to RunIt, the runner will scan the
// CLI's output for the sentinel value, before sending output to the Commander
// for processing.  When the sentinel value is found, the call to RunIt returns
// without error.  If the sentinel is not found before the deadline, RunIt
// returns an error.
//
type ProcRunner struct {
	params      *Parameters     // specifics about a particular CLI
	cmd         *exec.Cmd       // the CLI subprocess
	stdIn       io.WriteCloser  // the CLI's input stream
	outScanner  *bufio.Scanner  // scans the CLI's standard output
	errScanner  *bufio.Scanner  // scans the CLI's error output
	chOut       chan []byte     // lines from stdOut
	chErr       chan []byte     // lines from stdErr
	infraErrors *errorTracker   // multiple threads can generate errors
	mutexState  sync.Mutex      // protect the ProcRunner state
	filter      *sentinelFilter // runs commands and watches for sentinels
}

type runnerState int

type logSink struct{}

var DebugMode = false

func (l logSink) Write(p []byte) (n int, err error) {
	if DebugMode {
		return fmt.Fprint(os.Stderr, string(p))
	}
	return 0, nil
}

var logger = log.New(&logSink{}, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

const (
	// Construction parameters are okay, but no subprocess running.
	// In this state after a call to NewProcRunner or Close.
	// Can change to stateError or stateIdle.
	stateUninitialized runnerState = iota

	// Ready for a command.
	// Can change to any other state.
	stateIdle

	// A command is Running.
	// Can change to stateError or stateIdle.
	stateRunning

	// Unrecoverable error, e.g. subprocess timed out on last command and
	// might be hung.
	// Cannot change to another state; the ProcRunner no longer usable.
	stateError
)

// lastError reports the most recent error.
func (pr *ProcRunner) lastError() error {
	return pr.infraErrors.lastError()
}

func (pr *ProcRunner) getState() runnerState {
	if pr.lastError() != nil {
		return stateError
	}
	if pr.cmd == nil {
		return stateUninitialized
	}
	if pr.filter.isRunning() {
		return stateRunning
	}
	return stateIdle
}

func (pr *ProcRunner) enterStateError(err error) {
	if err == nil {
		panic("cannot enter error state w/o an error")
	}
	pr.infraErrors.log(err)
}

func (pr *ProcRunner) enterStateUninitialized() {
	pr.cmd = nil
}

// NewProcRunner returns a new ProcRunner, or an error on bad parameters.
func NewProcRunner(params *Parameters) (*ProcRunner, error) {
	logger.Println("creating new ProcRunner")
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return &ProcRunner{
		params: params,
		filter: makeSentinelFilter(
			params.OutSentinel, params.ErrSentinel, params.CommandTerminator),
	}, nil
}

// RunIgnoringOutput runs the given command ignoring its output.
// A default timeout is used.
func (pr *ProcRunner) RunIgnoringOutput(c string) error {
	return pr.RunIt(&cmdrs.KondoCommander{Command: c}, 0)
}

// RunIt runs the given Commander in the given duration.
//
// RunIt blocks until the command completes, or the duration passes. After a
// call to RunIt returns, with or without an error, the Commander may be
// consulted for data it accumulated. If RunIt returned an error, the Commander
// might not have complete results.
//
// RunIt returns an error from either the Commander or from ProcRunner's own
// internal infrastructure, e.g. a timeout.  The Commander should _not_ return
// an error on some minor parsing trouble - instead it should note the error
// internally for later reporting to whatever owns it, and return no error to
// the ProcRunner. A Commander should only return an error to the ProcRunner in
// the rare case that it (the Commander) determines that the subprocess should
// no longer be used by itself or any other Commander.
//
// If RunIt returns an error, then the ProcRunner should be abandoned.
// There's no general way to interrupt and "fix" a subprocess.
func (pr *ProcRunner) RunIt(cmdr Commander, timeOut time.Duration) error {
	// Don't defer the 'Unlock' call corresponding to this Lock.
	// We must unlock well before exiting this function because we intend to run
	// a potentially long-running command.
	logger.Printf("beginning RunIt for command %q\n", cmdr.String())
	pr.mutexState.Lock()
	switch pr.getState() {
	case stateError:
		logger.Println("entering state error")
		pr.mutexState.Unlock()
		return fmt.Errorf("subprocess in error state, cannot recover")
	case stateRunning:
		logger.Println("already running")
		pr.mutexState.Unlock()
		return fmt.Errorf("already running something")
	case stateUninitialized:
		logger.Println("in state uninitialized")
		if err := pr.startSubprocess(); err != nil {
			pr.enterStateError(err)
			pr.mutexState.Unlock()
			return err
		}
		// immediately enter stateIdle and do the run
		fallthrough
	case stateIdle:
		logger.Println("in state idle, starting run")
		if cmdr == nil {
			pr.mutexState.Unlock()
			return fmt.Errorf("provide a Commander")
		}
		// enter stateRunning
		logger.Println("entering state running")
		_, err := pr.filter.BeginRun(cmdr, pr.stdIn)
		pr.mutexState.Unlock()
		if err != nil {
			return err
		}
		// The following call should consume no more than "timeOut" wall clock time.
		if err = pr.filter.IssueSentinelsAndFilter(
			pr.chOut, pr.chErr, timeOut); err != nil {
			pr.enterStateError(err)
			return err
		}
		// exit stateRunning, back to stateIdle.
		// This relies on sentinelFilter working as expected.
		return nil
	default:
		pr.mutexState.Unlock()
		return fmt.Errorf("unknown state %d", pr.getState())
	}
}

// startSubprocess starts the CLI subprocess, returning an error on any trouble.
func (pr *ProcRunner) startSubprocess() (err error) {
	pr.infraErrors = &errorTracker{}

	pr.cmd = exec.Command(pr.params.Path, pr.params.Args...)
	pr.cmd.Dir = pr.params.WorkingDir

	// Set up pipes and buffered scanners.
	if err = pr.setUpPipesAndScanners(); err != nil {
		return err
	}

	logger.Printf("starting subprocess: %q\n", pr.cmd.String())

	// Assure that the subprocess is started without error before
	// doing anything else.
	// The I/O pipes for the subprocess are buffered; it can wait.
	if err = pr.cmd.Start(); err != nil {
		return fmt.Errorf("trying to start %s - %w", pr.params.Path, err)
	}

	logger.Printf("seems to have started ok\n")
	// Scan the subprocess' output.
	// Send its stdErr and stdOut to a combined output channel.
	// There might be lots of output, so buffer the channel.
	// The number corresponds to the number of lines.
	pr.chOut = make(chan []byte, 10000)
	pr.chErr = make(chan []byte, 10)
	var scanWg sync.WaitGroup
	scanWg.Add(2)
	go pr.scanStdErr(&scanWg)
	go pr.scanStdOut(&scanWg)

	// Wait for completion of both scanners.  They should complete on subprocess
	// exit, regardless of exit code. If the subprocess fails to close its stdErr
	// and stdOut, this will hang, and chOut won't close.  The client is
	// protected from this hang by the timeout sent into RunIt.
	go func() {
		logger.Println("waiting for subprocess exit")

		waitErr := pr.cmd.Wait()
		// find out at runtime if this is true by checking second value

		logger.Println("subprocess finished")
		if exitErr, isExitError := waitErr.(*exec.ExitError); isExitError {
			logger.Println("detected exit error: " + exitErr.Error())
			pr.enterStateError(
				errors.Wrap(exitErr, "subprocess exited with err"))
		} else if waitErr != nil {
			logger.Println("encounter some error other than exit failure")
			pr.enterStateError(
				errors.Wrap(exitErr, "subprocess erred out"))
		}
		// The following should end quickly if cmd.Wait worked.
		scanWg.Wait()
		// We're all done with this subprocess.
		// Close the channel to shut down parsing.
		close(pr.chOut)
		pr.enterStateUninitialized()
	}()
	return nil
}

// Close gracefully terminates the CLI, and shuts down all streams, reporting
// any errors that happen.
//
// Close sends the CLI's ExitCommand (if not empty) and EOF, and returns the
// process' exit code in string form.  If the exit code was 0, nil is returned.
//
// TODO: kill a hung process, make it possible to transition from
// stateError to stateUninitialized.
func (pr *ProcRunner) Close() (err error) {
	pr.mutexState.Lock()
	defer pr.mutexState.Unlock()
	switch pr.getState() {
	case stateUninitialized:
		return nil
	case stateRunning:
		return fmt.Errorf("cannot interrupt run")
	case stateError:
		return fmt.Errorf("cannot close error state")
	case stateIdle:
		return pr.attemptShutdown()
	default:
		return fmt.Errorf("unknown close state %d", pr.getState())
	}
}

func (pr *ProcRunner) attemptShutdown() error {
	// This is a no-op if the exit command is empty.
	if _, err := pr.filter.BeginRun(
		&cmdrs.KondoCommander{Command: pr.params.ExitCommand},
		pr.stdIn); err != nil {
		pr.enterStateError(err)
		return err
	}
	// The following is like sending an EOF on the input, and should trigger
	// shutdown of the scanners on stdErr and stdOut.
	if err := pr.stdIn.Close(); err != nil {
		pr.enterStateError(err)
		return err
	}
	return nil
}

// setUpPipesAndScanners establishes the necessary pipes.
func (pr *ProcRunner) setUpPipesAndScanners() (err error) {
	pr.stdIn, err = pr.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("getting stdIn for %q; %w", pr.params.Path, err)
	}
	var pipe io.ReadCloser
	pipe, err = pr.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("getting stdOut for %q; %w", pr.params.Path, err)
	}
	pr.outScanner = bufio.NewScanner(pipe)
	pipe, err = pr.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("getting stdErr for %q; %w", pr.params.Path, err)
	}
	pr.errScanner = bufio.NewScanner(pipe)
	return nil
}

func (pr *ProcRunner) scanStdErr(wg *sync.WaitGroup) {
	defer wg.Done()
	if len(pr.params.ErrPrefix) > 0 {
		for pr.errScanner.Scan() {
			var buff bytes.Buffer
			buff.WriteString(pr.params.ErrPrefix)
			buff.Write(pr.errScanner.Bytes())
			pr.chErr <- buff.Bytes()
		}
	} else {
		for pr.errScanner.Scan() {
			line := pr.errScanner.Bytes()
			send := make([]byte, len(line))
			copy(send, line)
			pr.chErr <- send
		}
	}
	if err := pr.errScanner.Err(); err != nil {
		// This should be rare.
		pr.enterStateError(fmt.Errorf("errScanner saw : %w", err))
	}
}

func (pr *ProcRunner) scanStdOut(wg *sync.WaitGroup) {
	defer wg.Done()
	logger.Println("Entered scanStdOut")
	count := 0
	for pr.outScanner.Scan() {
		line := pr.outScanner.Bytes()
		count++
		logger.Printf("Managed to read line: %s\n", string(line))
		send := make([]byte, len(line))
		copy(send, line)
		pr.chOut <- send
	}
	logger.Printf("scanStdOut ended, read %d lines!\n", count)
	if err := pr.outScanner.Err(); err != nil {
		// This should be rare.
		logger.Printf("scanStdOut 'rare' error was %s!\n", err.Error())
		pr.enterStateError(fmt.Errorf("outScanner saw : %w", err))
	}
}
