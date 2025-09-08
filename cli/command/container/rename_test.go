package container

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"gotest.tools/v3/assert"
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
			expectedErr: "invalid container name or ID: value is empty",
		},
		{
			doc:         "empty new name",
			oldName:     "oldName",
			newName:     "",
			expectedErr: "new name cannot be blank",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.doc, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{
				containerRenameFunc: func(ctx context.Context, oldName, newName string) error {
					if oldName == "" {
						return errors.New("invalid container name or ID: value is empty")
					}
					return nil
				},
			})

			cmd := newRenameCommand(cli)
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
