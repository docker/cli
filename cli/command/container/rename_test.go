package container

import (
	"context"
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestRunRename(t *testing.T) {
	testcases := []struct {
		doc, oldName, newName, expectedErr string
	}{
		{
			doc:         "success",
			oldName:     "oldName",
			newName:     "newName",
			expectedErr: "",
		},
		{
			doc:         "empty old name",
			oldName:     "",
			newName:     "newName",
			expectedErr: "Error: Neither old nor new names may be empty",
		},
		{
			doc:         "empty new name",
			oldName:     "oldName",
			newName:     "",
			expectedErr: "Error: Neither old nor new names may be empty",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.doc, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{
				containerRenameFunc: func(ctx context.Context, oldName, newName string) error {
					return nil
				},
			})

			cmd := NewRenameCommand(cli)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs([]string{tc.oldName, tc.newName})

			err := cmd.Execute()

			if tc.expectedErr != "" {
				assert.ErrorContains(t, err, tc.expectedErr)
			} else {
				assert.NilError(t, err)
			}
		})
	}
}

func TestRunRenameClientError(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		containerRenameFunc: func(ctx context.Context, oldName, newName string) error {
			return errors.New("client error")
		},
	})

	cmd := NewRenameCommand(cli)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"oldName", "newName"})

	err := cmd.Execute()

	assert.Check(t, is.Error(err, "Error: failed to rename container named oldName"))
}
