package pruner

import (
	"errors"
	"io"
	"testing"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/streams"
	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
)

type versionedCLI struct {
	command.Cli
	version string
}

func (c versionedCLI) CurrentVersion() string { return c.version }

func TestProgressPrinter(t *testing.T) {
	for _, tc := range []struct {
		contentType ContentType
		expected    string
	}{
		{TypeContainer, "Deleted Containers:\nfirst\nsecond\n\n"},
		{TypeNetwork, "Deleted Networks:\nfirst\nsecond\n\n"},
		{TypeVolume, "Deleted Volumes:\nfirst\nsecond\n\n"},
		{TypeImage, "Deleted Images:\nuntagged: first\ndeleted: second\n\n"},
		{TypeBuildCache, "Deleted build cache objects:\nfirst\nsecond\n\n"},
	} {
		t.Run(string(tc.contentType), func(t *testing.T) {
			cli := test.NewFakeCli(nil)
			p := NewProgressPrinter(versionedCLI{Cli: cli, version: "1.56"}, tc.contentType)
			assert.Assert(t, p.OnProgress != nil)
			assert.NilError(t, p.OnProgress(client.PruneProgress{ID: "first", Action: "untagged"}))
			assert.Assert(t, p.Started)
			assert.NilError(t, p.OnProgress(client.PruneProgress{ID: "second", Action: "deleted"}))
			assert.NilError(t, p.Finish())
			assert.Equal(t, cli.OutBuffer().String(), tc.expected)
		})
	}
}

func TestProgressPrinterOldAPI(t *testing.T) {
	cli := test.NewFakeCli(nil)
	p := NewProgressPrinter(versionedCLI{Cli: cli, version: "1.55"}, TypeContainer)
	assert.Assert(t, p.OnProgress == nil)
	assert.NilError(t, p.Finish())
	assert.Equal(t, cli.OutBuffer().String(), "")
}

func TestProgressPrinterEmpty(t *testing.T) {
	cli := test.NewFakeCli(nil)
	p := NewProgressPrinter(versionedCLI{Cli: cli, version: "1.56"}, TypeContainer)
	assert.NilError(t, p.Finish())
	assert.Equal(t, cli.OutBuffer().String(), "")
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestProgressPrinterOutputError(t *testing.T) {
	cli := test.NewFakeCli(nil)
	cli.SetOut(streams.NewOut(failingWriter{}))
	p := NewProgressPrinter(versionedCLI{Cli: cli, version: "1.56"}, TypeContainer)
	err := p.OnProgress(client.PruneProgress{ID: "first", Action: "deleted"})
	assert.Assert(t, errors.Is(err, io.ErrClosedPipe))
}
