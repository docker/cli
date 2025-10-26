package stack

import (
	"context"
	"io"
	"testing"

	"github.com/docker/cli/cli/compose/convert"
	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestDeployWithEmptyName(t *testing.T) {
	cmd := newDeployCommand(test.NewFakeCli(&fakeClient{}))
	cmd.SetArgs([]string{"'   '"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	assert.ErrorContains(t, cmd.Execute(), `invalid stack name: "'   '"`)
}

func TestPruneServices(t *testing.T) {
	ctx := context.Background()
	namespace := convert.NewNamespace("foo")
	services := map[string]struct{}{
		"new":  {},
		"keep": {},
	}
	apiClient := &fakeClient{services: []string{objectName("foo", "keep"), objectName("foo", "remove")}}
	dockerCli := test.NewFakeCli(apiClient)

	pruneServices(ctx, dockerCli, namespace, services)
	assert.Check(t, is.DeepEqual(buildObjectIDs([]string{objectName("foo", "remove")}), apiClient.removedServices))
}

// TestServiceUpdateResolveImageChanged tests that the service's
// image digest, and "ForceUpdate" is preserved if the image did not change in
// the compose file
func TestServiceUpdateResolveImageChanged(t *testing.T) {
	namespace := convert.NewNamespace("mystack")

	var receivedOptions client.ServiceUpdateOptions

	fakeCli := test.NewFakeCli(&fakeClient{
		serviceListFunc: func(options client.ServiceListOptions) (client.ServiceListResult, error) {
			return client.ServiceListResult{
				Items: []swarm.Service{
					{
						Spec: swarm.ServiceSpec{
							Annotations: swarm.Annotations{
								Name:   namespace.Name() + "_myservice",
								Labels: map[string]string{"com.docker.stack.image": "foobar:1.2.3"},
							},
							TaskTemplate: swarm.TaskSpec{
								ContainerSpec: &swarm.ContainerSpec{
									Image: "foobar:1.2.3@sha256:deadbeef",
								},
								ForceUpdate: 123,
							},
						},
					},
				},
			}, nil
		},
		serviceUpdateFunc: func(serviceID string, options client.ServiceUpdateOptions) (client.ServiceUpdateResult, error) {
			receivedOptions = options
			return client.ServiceUpdateResult{}, nil
		},
	})

	testcases := []struct {
		image                 string
		expectedQueryRegistry bool
		expectedImage         string
		expectedForceUpdate   uint64
	}{
		// Image not changed
		{
			image:                 "foobar:1.2.3",
			expectedQueryRegistry: false,
			expectedImage:         "foobar:1.2.3@sha256:deadbeef",
			expectedForceUpdate:   123,
		},
		// Image changed
		{
			image:                 "foobar:1.2.4",
			expectedQueryRegistry: true,
			expectedImage:         "foobar:1.2.4",
			expectedForceUpdate:   123,
		},
	}

	ctx := context.Background()

	for _, tc := range testcases {
		t.Run(tc.image, func(t *testing.T) {
			spec := map[string]swarm.ServiceSpec{
				"myservice": {
					TaskTemplate: swarm.TaskSpec{
						ContainerSpec: &swarm.ContainerSpec{
							Image: tc.image,
						},
					},
				},
			}
			_, err := deployServices(ctx, fakeCli, spec, namespace, false, resolveImageChanged)
			assert.NilError(t, err)
			assert.Check(t, is.Equal(receivedOptions.QueryRegistry, tc.expectedQueryRegistry))
			assert.Check(t, is.Equal(receivedOptions.Spec.TaskTemplate.ContainerSpec.Image, tc.expectedImage))
			assert.Check(t, is.Equal(receivedOptions.Spec.TaskTemplate.ForceUpdate, tc.expectedForceUpdate))

			receivedOptions = client.ServiceUpdateOptions{}
		})
	}
}
