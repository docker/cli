package stack

import (
	"context"
	"testing"

	"github.com/docker/cli/internal/test"
	composetypes "github.com/docker/stacks/pkg/compose/types"
	stacktypes "github.com/docker/stacks/pkg/types"
	"gotest.tools/assert"
)

func TestAuthHeaderForStackNoVars(t *testing.T) {
	ctx := context.Background()
	cli := test.NewFakeCli(&fakeClient{
		version: clientSideStackVersion,
	})
	stackCreate := &stacktypes.StackCreate{
		Spec: stacktypes.StackSpec{
			Services: composetypes.Services{
				composetypes.ServiceConfig{
					Image: "myregistry.com/acme/imagename:latest",
				},
			},
		},
	}

	_, err := getAuthHeaderForStack(ctx, cli, stackCreate)
	assert.NilError(t, err)
}
func TestAuthHeaderForStackWithVars(t *testing.T) {
	ctx := context.Background()
	cli := test.NewFakeCli(&fakeClient{
		version: clientSideStackVersion,
	})
	stackCreate := &stacktypes.StackCreate{
		Spec: stacktypes.StackSpec{
			Services: composetypes.Services{
				composetypes.ServiceConfig{
					Image: "${IMAGE}",
				},
			},
			PropertyValues: []string{"IMAGE=myregistry.com/acme/imagename:latest"},
		},
	}

	_, err := getAuthHeaderForStack(ctx, cli, stackCreate)
	assert.NilError(t, err)
}
