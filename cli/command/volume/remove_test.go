package volume

import (
	"io/ioutil"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/testutil"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestVolumeRemoveErrors(t *testing.T) {
	testCases := []struct {
		args             []string
		volumeRemoveFunc func(volumeID string, force bool) error
		expectedError    string
	}{
		{
			expectedError: "requires at least 1 argument",
		},
		{
			args: []string{"nodeID"},
			volumeRemoveFunc: func(volumeID string, force bool) error {
				return errors.Errorf("error removing the volume")
			},
			expectedError: "error removing the volume",
		},
	}
	for _, tc := range testCases {
		cmd := newRemoveCommand(
			test.NewFakeCli(&fakeClient{
				volumeRemoveFunc: tc.volumeRemoveFunc,
			}))
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		testutil.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestNodeRemoveMultiple(t *testing.T) {
	cmd := newRemoveCommand(test.NewFakeCli(&fakeClient{}))
	cmd.SetArgs([]string{"volume1", "volume2"})
	assert.NoError(t, cmd.Execute())
}
