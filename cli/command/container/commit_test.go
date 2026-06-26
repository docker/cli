package container

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestRunCommit(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		containerCommitFunc: func(ctx context.Context, ctr string, options client.ContainerCommitOptions) (client.ContainerCommitResult, error) {
			assert.Check(t, is.Equal(options.Author, "Author Name <author@name.com>"))
			assert.Check(t, is.DeepEqual(options.Changes, []string{"EXPOSE 80"}))
			assert.Check(t, is.Equal(options.Comment, "commit message"))
			assert.Check(t, is.Equal(options.NoPause, true))
			assert.Check(t, is.Equal(ctr, "container-id"))

			return client.ContainerCommitResult{ID: "image-id"}, nil
		},
	})

	cmd := newCommitCommand(cli)
	cmd.SetOut(io.Discard)
	cmd.SetArgs(
		[]string{
			"--author", "Author Name <author@name.com>",
			"--change", "EXPOSE 80",
			"--message", "commit message",
			"--no-pause",
			"container-id",
		},
	)

	err := cmd.Execute()
	assert.NilError(t, err)

	assert.Assert(t, is.Equal(cli.OutBuffer().String(), "image-id\n"))
}

func TestRunCommitClientError(t *testing.T) {
	clientError := errors.New("client error")

	cli := test.NewFakeCli(&fakeClient{
		containerCommitFunc: func(ctx context.Context, ctr string, options client.ContainerCommitOptions) (client.ContainerCommitResult, error) {
			return client.ContainerCommitResult{}, clientError
		},
	})

	cmd := newCommitCommand(cli)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"container-id"})

	err := cmd.Execute()
	assert.ErrorIs(t, err, clientError)
}
