package opts

import (
	"testing"

	"github.com/moby/moby/api/types/container"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestGpusOptAll(t *testing.T) {
	for _, testcase := range []string{
		"all",
		"-1",
		"count=all",
		"count=-1",
	} {
		var gpus GpuOpts
		assert.Check(t, gpus.Set(testcase))
		gpuReqs := gpus.Value()
		assert.Assert(t, is.Len(gpuReqs, 1))
		assert.Check(t, is.DeepEqual(gpuReqs[0], container.DeviceRequest{
			Count:        -1,
			Capabilities: [][]string{{"gpu"}},
			Options:      map[string]string{},
		}))
	}
}

func TestGpusOptInvalidCount(t *testing.T) {
	var gpus GpuOpts
	err := gpus.Set(`count=invalid-integer`)
	assert.Error(t, err, `invalid count (invalid-integer): value must be either "all" or an integer: invalid syntax`)
}

func TestGpusOpts(t *testing.T) {
	for _, testcase := range []string{
		`driver=nvidia,"capabilities=compute,utility","options=foo=bar,baz=qux"`,
		`1,driver=nvidia,"capabilities=compute,utility","options=foo=bar,baz=qux"`,
		`count=1,driver=nvidia,"capabilities=compute,utility","options=foo=bar,baz=qux"`,
		`driver=nvidia,"capabilities=compute,utility","options=foo=bar,baz=qux",count=1`,
	} {
		var gpus GpuOpts
		assert.Check(t, gpus.Set(testcase))
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

func TestGpusOptDuplicateCount(t *testing.T) {
	for _, value := range []string{"count=1,2", "1,2", "all,1", "count=1,all", "1,count=2", "count=1,count=2"} {
		t.Run(value, func(t *testing.T) {
			var gpus GpuOpts
			assert.Error(t, gpus.Set(value), "gpu request key 'count' can be specified only once")
			assert.Assert(t, is.Len(gpus.Value(), 0))
		})
	}
}
