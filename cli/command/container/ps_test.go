package container

import (
	"testing"

	"github.com/docker/cli/opts"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
)

func TestBuildContainerListOptions(t *testing.T) {
	filters := opts.NewFilterOpt()
	assert.NilError(t, filters.Set("foo=bar"))
	assert.NilError(t, filters.Set("baz=foo"))

	contexts := []struct {
		psOpts          *psOptions
		expectedAll     bool
		expectedSize    bool
		expectedLimit   int
		expectedFilters map[string]string
	}{
		{
			psOpts: &psOptions{
				all:    true,
				size:   true,
				last:   5,
				filter: filters,
			},
			expectedAll:   true,
			expectedSize:  true,
			expectedLimit: 5,
			expectedFilters: map[string]string{
				"foo": "bar",
				"baz": "foo",
			},
		},
		{
			psOpts: &psOptions{
				all:     true,
				size:    true,
				last:    -1,
				nLatest: true,
			},
			expectedAll:     true,
			expectedSize:    true,
			expectedLimit:   1,
			expectedFilters: make(map[string]string),
		},
		{
			psOpts: &psOptions{
				all:    true,
				size:   false,
				last:   5,
				filter: filters,
				// With .Size, size should be true
				format: "{{.Size}}",
			},
			expectedAll:   true,
			expectedSize:  true,
			expectedLimit: 5,
			expectedFilters: map[string]string{
				"foo": "bar",
				"baz": "foo",
			},
		},
		{
			psOpts: &psOptions{
				all:    true,
				size:   false,
				last:   5,
				filter: filters,
				// With .Size, size should be true
				format: "{{.Size}} {{.CreatedAt}} {{.Networks}}",
			},
			expectedAll:   true,
			expectedSize:  true,
			expectedLimit: 5,
			expectedFilters: map[string]string{
				"foo": "bar",
				"baz": "foo",
			},
		},
		{
			psOpts: &psOptions{
				all:    true,
				size:   false,
				last:   5,
				filter: filters,
				// Without .Size, size should be false
				format: "{{.CreatedAt}} {{.Networks}}",
			},
			expectedAll:   true,
			expectedSize:  false,
			expectedLimit: 5,
			expectedFilters: map[string]string{
				"foo": "bar",
				"baz": "foo",
			},
		},
	}

	for _, c := range contexts {
		options, err := buildContainerListOptions(c.psOpts)
		assert.NilError(t, err)

		assert.Check(t, is.Equal(c.expectedAll, options.All))
		assert.Check(t, is.Equal(c.expectedSize, options.Size))
		assert.Check(t, is.Equal(c.expectedLimit, options.Limit))
		assert.Check(t, is.Equal(len(c.expectedFilters), options.Filters.Len()))

		for k, v := range c.expectedFilters {
			f := options.Filters
			if !f.ExactMatch(k, v) {
				t.Fatalf("Expected filter with key %s to be %s but got %s", k, v, f.Get(k))
			}
		}
	}
}
