package opts

import (
	"testing"

	"github.com/docker/docker/api/types/container"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func TestGpusOptAll(t *testing.T) {
	for _, testcase := range []string{
		"all",
		"-1",
		"count=all",
		"count=-1",
	} {
		var gpus GpuOpts
		gpus.Set(testcase)
		gpuReqs := gpus.Value()
		assert.Assert(t, is.Len(gpuReqs, 1))
		assert.Check(t, is.DeepEqual(gpuReqs[0], container.DeviceRequest{
			Count:        -1,
			Capabilities: [][]string{{"gpu"}},
			Options:      map[string]string{},
		}))
	}
}

func TestGpusOpts(t *testing.T) {
	for _, testcase := range []string{
		"driver=nvidia,\"capabilities=compute,utility\",\"options=foo=bar,baz=qux\"",
		"1,driver=nvidia,\"capabilities=compute,utility\",\"options=foo=bar,baz=qux\"",
		"count=1,driver=nvidia,\"capabilities=compute,utility\",\"options=foo=bar,baz=qux\"",
		"driver=nvidia,\"capabilities=compute,utility\",\"options=foo=bar,baz=qux\",count=1",
	} {
		var gpus GpuOpts
		gpus.Set(testcase)
		gpuReqs := gpus.Value()
		assert.Assert(t, is.Len(gpuReqs, 1))
		assert.Check(t, is.DeepEqual(gpuReqs[0], container.DeviceRequest{
			Driver:       "nvidia",
			Count:        1,
			Capabilities: [][]string{{"compute", "utility", "gpu"}},
			Options:      map[string]string{"foo": "bar", "baz": "qux"},
		}))
	}
}
