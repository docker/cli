package container

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types/container"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestRunCommit(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		containerCommitFunc: func(
			ctx context.Context,
			ctr string,
			options container.CommitOptions,
		) (container.CommitResponse, error) {
			assert.Check(t, is.Equal(options.Author, "Author Name <author@name.com>"))
			assert.Check(t, is.DeepEqual(options.Changes, []string{"EXPOSE 80"}))
			assert.Check(t, is.Equal(options.Comment, "commit message"))
			assert.Check(t, is.Equal(options.Pause, false))
			assert.Check(t, is.Equal(ctr, "container-id"))

			return container.CommitResponse{ID: "image-id"}, nil
		},
	})

	cmd := NewCommitCommand(cli)
	cmd.SetOut(io.Discard)
	cmd.SetArgs(
		[]string{
			"--author", "Author Name <author@name.com>",
			"--change", "EXPOSE 80",
			"--message", "commit message",
			"--pause=false",
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
		containerCommitFunc: func(
			ctx context.Context,
			ctr string,
			options container.CommitOptions,
		) (container.CommitResponse, error) {
			return container.CommitResponse{}, clientError
		},
	})

	cmd := NewCommitCommand(cli)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"container-id"})

	err := cmd.Execute()
	assert.ErrorIs(t, err, clientError)
}
