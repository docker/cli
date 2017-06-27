package service

import (
	"bytes"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestWarnDetachDefault(t *testing.T) {
	var detach bool
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	addDetachFlag(flags, &detach)

	var tests = []struct {
		detach  bool
		version string

		expectWarning bool
	}{
		{true, "1.28", false},
		{true, "1.29", false},
		{false, "1.28", false},
		{false, "1.29", true},
	}

	for _, test := range tests {
		out := new(bytes.Buffer)
		flags.Lookup(flagDetach).Changed = test.detach

		warnDetachDefault(out, test.version, flags, "")

		if test.expectWarning {
			assert.NotEmpty(t, out.String(), "expected warning")
		} else {
			assert.Empty(t, out.String(), "expected no warning")
		}
	}
}
