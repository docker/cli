package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/swarm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemBytesString(t *testing.T) {
	var mem opts.MemBytes = 1048576
	assert.Equal(t, "1MiB", mem.String())
}

func TestMemBytesSetAndValue(t *testing.T) {
	var mem opts.MemBytes
	assert.NoError(t, mem.Set("5kb"))
	assert.Equal(t, int64(5120), mem.Value())
}

func TestNanoCPUsString(t *testing.T) {
	var cpus opts.NanoCPUs = 6100000000
	assert.Equal(t, "6.100", cpus.String())
}

func TestNanoCPUsSetAndValue(t *testing.T) {
	var cpus opts.NanoCPUs
	assert.NoError(t, cpus.Set("0.35"))
	assert.Equal(t, int64(350000000), cpus.Value())
}

func TestUint64OptString(t *testing.T) {
	value := uint64(2345678)
	opt := Uint64Opt{value: &value}
	assert.Equal(t, "2345678", opt.String())

	opt = Uint64Opt{}
	assert.Equal(t, "", opt.String())
}

func TestUint64OptSetAndValue(t *testing.T) {
	var opt Uint64Opt
	assert.NoError(t, opt.Set("14445"))
	assert.Equal(t, uint64(14445), *opt.Value())
}

func TestHealthCheckOptionsToHealthConfig(t *testing.T) {
	dur := time.Second
	opt := healthCheckOptions{
		cmd:         "curl",
		interval:    opts.PositiveDurationOpt{*opts.NewDurationOpt(&dur)},
		timeout:     opts.PositiveDurationOpt{*opts.NewDurationOpt(&dur)},
		startPeriod: opts.PositiveDurationOpt{*opts.NewDurationOpt(&dur)},
		retries:     10,
	}
	config, err := opt.toHealthConfig()
	assert.NoError(t, err)
	assert.Equal(t, &container.HealthConfig{
		Test:        []string{"CMD-SHELL", "curl"},
		Interval:    time.Second,
		Timeout:     time.Second,
		StartPeriod: time.Second,
		Retries:     10,
	}, config)
}

func TestHealthCheckOptionsToHealthConfigNoHealthcheck(t *testing.T) {
	opt := healthCheckOptions{
		noHealthcheck: true,
	}
	config, err := opt.toHealthConfig()
	assert.NoError(t, err)
	assert.Equal(t, &container.HealthConfig{
		Test: []string{"NONE"},
	}, config)
}

func TestHealthCheckOptionsToHealthConfigConflict(t *testing.T) {
	opt := healthCheckOptions{
		cmd:           "curl",
		noHealthcheck: true,
	}
	_, err := opt.toHealthConfig()
	assert.EqualError(t, err, "--no-healthcheck conflicts with --health-* options")
}

func TestResourceOptionsToResourceRequirements(t *testing.T) {
	incorrectOptions := []resourceOptions{
		{
			resGenericResources: []string{"foo=bar", "foo=1"},
		},
		{
			resGenericResources: []string{"foo=bar", "foo=baz"},
		},
		{
			resGenericResources: []string{"foo=bar"},
		},
		{
			resGenericResources: []string{"foo=1", "foo=2"},
		},
	}

	for _, opt := range incorrectOptions {
		_, err := opt.ToResourceRequirements()
		assert.Error(t, err)
	}

	correctOptions := []resourceOptions{
		{
			resGenericResources: []string{"foo=1"},
		},
		{
			resGenericResources: []string{"foo=1", "bar=2"},
		},
	}

	for _, opt := range correctOptions {
		r, err := opt.ToResourceRequirements()
		assert.NoError(t, err)
		assert.Len(t, r.Reservations.GenericResources, len(opt.resGenericResources))
	}

}

func TestToServiceNetwork(t *testing.T) {
	nws := []types.NetworkResource{
		{Name: "aaa-network", ID: "id555"},
		{Name: "mmm-network", ID: "id999"},
		{Name: "zzz-network", ID: "id111"},
	}

	client := &fakeClient{
		networkInspectFunc: func(ctx context.Context, networkID string, options types.NetworkInspectOptions) (types.NetworkResource, error) {
			for _, network := range nws {
				if network.ID == networkID || network.Name == networkID {
					return network, nil
				}
			}
			return types.NetworkResource{}, fmt.Errorf("network not found: %s", networkID)
		},
	}

	nwo := opts.NetworkOpt{}
	nwo.Set("zzz-network")
	nwo.Set("mmm-network")
	nwo.Set("aaa-network")

	o := newServiceOptions()
	o.mode = "replicated"
	o.networks = nwo

	ctx := context.Background()
	flags := newCreateCommand(nil).Flags()
	service, err := o.ToService(ctx, client, flags)
	require.NoError(t, err)
	assert.Equal(t, []swarm.NetworkAttachmentConfig{{Target: "id111"}, {Target: "id555"}, {Target: "id999"}}, service.TaskTemplate.Networks)
}
