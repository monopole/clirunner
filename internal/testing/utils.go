package testing

import (
	"github.com/stretchr/testify/assert"
	"sort"
	"strings"
	"testing"
)

// AssertEqualAnyOrder returns true if the two strings, viewed as lines,
// are equal after both are sorted.
func AssertEqualAnyOrder(t *testing.T, s1 string, s2 string) {
	lines1 := strings.Split(s1, "\n")
	sort.Strings(lines1)
	lines2 := strings.Split(s2, "\n")
	sort.Strings(lines2)
	assert.Equal(t, lines1, lines2)
}
