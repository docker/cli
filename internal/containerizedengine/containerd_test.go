package containerizedengine

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/containerd/containerd"
	"github.com/docker/cli/cli/streams"
	"github.com/docker/docker/api/types"
	"gotest.tools/assert"
)

func TestPullWithAuthPullFail(t *testing.T) {
	ctx := context.Background()
	client := baseClient{
		cclient: &fakeContainerdClient{
			pullFunc: func(ctx context.Context, ref string, opts ...containerd.RemoteOpt) (containerd.Image, error) {
				return nil, fmt.Errorf("pull failure")

			},
		},
	}
	imageName := "testnamegoeshere"

	_, err := client.pullWithAuth(ctx, imageName, streams.NewOut(&bytes.Buffer{}), &types.AuthConfig{})
	assert.ErrorContains(t, err, "pull failure")
}

func TestPullWithAuthPullPass(t *testing.T) {
	ctx := context.Background()
	client := baseClient{
		cclient: &fakeContainerdClient{
			pullFunc: func(ctx context.Context, ref string, opts ...containerd.RemoteOpt) (containerd.Image, error) {
				return nil, nil

			},
		},
	}
	imageName := "testnamegoeshere"

	_, err := client.pullWithAuth(ctx, imageName, streams.NewOut(&bytes.Buffer{}), &types.AuthConfig{})
	assert.NilError(t, err)
}
