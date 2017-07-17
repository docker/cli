// +build daemon

package main

import (
	"io/ioutil"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func stubRun(cmd *cobra.Command, args []string) error {
	return nil
}

func TestDaemonCommandHelp(t *testing.T) {
	cmd := newDaemonCommand()
	cmd.RunE = stubRun
	cmd.SetArgs([]string{"--help"})
	cmd.SetOutput(ioutil.Discard)
	err := cmd.Execute()
	assert.NoError(t, err)
}

func TestDaemonCommand(t *testing.T) {
	cmd := newDaemonCommand()
	cmd.RunE = stubRun
	cmd.SetArgs([]string{"--containerd", "/foo"})
	cmd.SetOutput(ioutil.Discard)
	err := cmd.Execute()
	assert.NoError(t, err)
}
