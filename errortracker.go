package clirunner

import "sync"

// errorTracker accumulates errors for debugging and reporting.
type errorTracker struct {
	m    sync.Mutex
	errs []error
}

func (et *errorTracker) log(err error) {
	if err == nil {
		return
	}
	et.m.Lock()
	et.errs = append(et.errs, err)
	et.m.Unlock()
}

func (et *errorTracker) lastError() error {
	if et == nil || len(et.errs) == 0 {
		return nil
	}
	return et.errs[len(et.errs)-1]
}
