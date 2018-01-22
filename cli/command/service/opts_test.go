package service

import (
	"testing"
	"time"

	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types/container"
	"github.com/spf13/pflag"
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

func durationPtr(duration time.Duration) *time.Duration {
	return &duration
}

func TestDetachOptSet(t *testing.T) {
	var testcases = []struct {
		value       string
		expectedErr string
		expected    detachOpt
	}{
		{value: "true", expected: detachOpt{immediate: true}},
		{value: "false", expected: detachOpt{}},
		{
			value:    "10s",
			expected: detachOpt{timeout: durationPtr(10 * time.Second)},
		},
		{
			value:       "invalid",
			expectedErr: "invalid bool or duration: invalid",
		},
		{
			value:    "after=20s",
			expected: detachOpt{timeout: durationPtr(20 * time.Second)},
		},
		{
			value:       "after=",
			expectedErr: "invalid bool or duration: after=",
		},
	}

	for _, testcase := range testcases {
		opt := detachOpt{}
		err := opt.Set(testcase.value)
		if testcase.expectedErr == "" {
			assert.NoError(t, err)
			assert.Equal(t, testcase.expected, opt)
		} else {
			assert.EqualError(t, err, testcase.expectedErr)
		}
	}
}

func TestDetachOptString(t *testing.T) {
	var testcases = []struct {
		value    string
		expected string
	}{
		{value: "true", expected: "true"},
		{value: "false", expected: "false"},
		{value: "10s", expected: "timeout=10s"},
		{value: "2m", expected: "timeout=2m0s"},
	}

	for _, testcase := range testcases {
		opt := &detachOpt{}
		if assert.NoError(t, opt.Set(testcase.value)) {
			assert.Equal(t, testcase.expected, opt.String())
		}
	}
}

func TestAddDetachFlagDefaultFlagValue(t *testing.T) {
	flags := pflag.NewFlagSet("testing", pflag.ContinueOnError)
	detach := detachOpt{}
	addDetachFlag(flags, &detach)

	err := flags.Parse([]string{"--detach"})
	require.NoError(t, err)
	assert.Equal(t, detachOpt{immediate: true}, detach)
}
