package swarm

import (
	"context"
	"testing"

	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/compose/convert"
	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
)

func TestPruneServices(t *testing.T) {
	ctx := context.Background()
	namespace := convert.NewNamespace("foo")
	services := map[string]struct{}{
		"new":  {},
		"keep": {},
	}
	client := &fakeClient{services: []string{objectName("foo", "keep"), objectName("foo", "remove")}}
	dockerCli := test.NewFakeCli(client)

	pruneServices(ctx, dockerCli, namespace, services)
	assert.Check(t, is.DeepEqual(buildObjectIDs([]string{objectName("foo", "remove")}), client.removedServices))
}

func TestDeployWithEmptyName(t *testing.T) {
	ctx := context.Background()
	client := &fakeClient{}
	dockerCli := test.NewFakeCli(client)

	err := deployCompose(ctx, dockerCli, options.Deploy{Namespace: "'   '", Prune: true})
	assert.Check(t, is.Error(err, `invalid stack name: "'   '"`))
}

// TestServiceUpdateResolveImageChanged tests that the service's
// image digest, and "ForceUpdate" is preserved if the image did not change in
// the compose file
func TestServiceUpdateResolveImageChanged(t *testing.T) {
	namespace := convert.NewNamespace("mystack")

	var (
		receivedOptions types.ServiceUpdateOptions
		receivedService swarm.ServiceSpec
	)

	client := test.NewFakeCli(&fakeClient{
		serviceListFunc: func(options types.ServiceListOptions) ([]swarm.Service, error) {
			return []swarm.Service{
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
			}, nil
		},
		serviceUpdateFunc: func(serviceID string, version swarm.Version, service swarm.ServiceSpec, options types.ServiceUpdateOptions) (types.ServiceUpdateResponse, error) {
			receivedOptions = options
			receivedService = service
			return types.ServiceUpdateResponse{}, nil
		},
	})

	var testcases = []struct {
		image                 string
		expectedQueryRegistry bool
		expectedImage         string
		expectedForceUpdate   uint64
	}{
		// Image not changed
		{
			image: "foobar:1.2.3",
			expectedQueryRegistry: false,
			expectedImage:         "foobar:1.2.3@sha256:deadbeef",
			expectedForceUpdate:   123,
		},
		// Image changed
		{
			image: "foobar:1.2.4",
			expectedQueryRegistry: true,
			expectedImage:         "foobar:1.2.4",
			expectedForceUpdate:   123,
		},
	}

	ctx := context.Background()

	for _, testcase := range testcases {
		t.Logf("Testing image %q", testcase.image)
		spec := map[string]swarm.ServiceSpec{
			"myservice": {
				TaskTemplate: swarm.TaskSpec{
					ContainerSpec: &swarm.ContainerSpec{
						Image: testcase.image,
					},
				},
			},
		}
		err := deployServices(ctx, client, spec, namespace, false, ResolveImageChanged)
		assert.NilError(t, err)
		assert.Check(t, is.Equal(receivedOptions.QueryRegistry, testcase.expectedQueryRegistry))
		assert.Check(t, is.Equal(receivedService.TaskTemplate.ContainerSpec.Image, testcase.expectedImage))
		assert.Check(t, is.Equal(receivedService.TaskTemplate.ForceUpdate, testcase.expectedForceUpdate))

		receivedService = swarm.ServiceSpec{}
		receivedOptions = types.ServiceUpdateOptions{}
	}
}
