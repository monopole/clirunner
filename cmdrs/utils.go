package cmdrs

import (
	"io"
)

// WriteString writes to the Writer, discarding the returned byte count.
// The error is always non-nil if the returned count != len(s), so the count
// isn't interesting if all you care about is error existence.
// Handy in test assertions, e.g. assert.NoError(t, WriteString(w, "data"))
func WriteString(w io.Writer, s string) (err error) {
	_, err = io.WriteString(w, s)
	return
}
