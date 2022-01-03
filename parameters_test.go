package clirunner_test

import (
	"testing"

	. "github.com/monopole/clirunner/cmdrs"

	. "github.com/monopole/clirunner"
	"github.com/stretchr/testify/assert"
)

func TestParameters_Validate(t *testing.T) {
	p := Parameters{}
	err := p.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must specify a Path")

	p.Path = "/whatever"
	err = p.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must specify OutSentinel")

	p.OutSentinel = &SimpleSentinelCommander{}
	err = p.Validate()
	assert.NoError(t, err)
}
