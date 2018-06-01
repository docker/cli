package kubernetes

import (
	"testing"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/assert/cmp"
)

func TestServiceFiltersLabelSelectorGen(t *testing.T) {
	cases := []struct {
		name                  string
		stackName             string
		filters               filters.Args
		expectedSelectorParts []string
	}{
		{
			name:      "no-filter",
			stackName: "test",
			filters:   filters.NewArgs(),
			expectedSelectorParts: []string{
				"com.docker.stack.namespace=test",
			},
		},
		{
			name:      "label present filter",
			stackName: "test",
			filters: filters.NewArgs(
				filters.KeyValuePair{Key: "label", Value: "label-is-present"},
			),
			expectedSelectorParts: []string{
				"com.docker.stack.namespace=test",
				"label-is-present",
			},
		},
		{
			name:      "single value label filter",
			stackName: "test",
			filters: filters.NewArgs(
				filters.KeyValuePair{Key: "label", Value: "label1=test"},
			),
			expectedSelectorParts: []string{
				"com.docker.stack.namespace=test",
				"label1=test",
			},
		},
		{
			name:      "multi value label filter",
			stackName: "test",
			filters: filters.NewArgs(
				filters.KeyValuePair{Key: "label", Value: "label1=test"},
				filters.KeyValuePair{Key: "label", Value: "label1=test2"},
			),
			expectedSelectorParts: []string{
				"com.docker.stack.namespace=test",
				"label1=test",
				"label1=test2",
			},
		},
		{
			name:      "2 different labels filter",
			stackName: "test",
			filters: filters.NewArgs(
				filters.KeyValuePair{Key: "label", Value: "label1=test"},
				filters.KeyValuePair{Key: "label", Value: "label2=test2"},
			),
			expectedSelectorParts: []string{
				"com.docker.stack.namespace=test",
				"label1=test",
				"label2=test2",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result := generateLabelSelector(c.filters, c.stackName)
			for _, toFind := range c.expectedSelectorParts {
				assert.Assert(t, cmp.Contains(result, toFind))
			}
		})
	}
}
func TestServiceFiltersServiceByName(t *testing.T) {
	cases := []struct {
		name             string
		filters          []string
		services         []swarm.Service
		expectedServices []swarm.Service
	}{
		{
			name:             "no filter",
			filters:          []string{},
			services:         makeServices("s1", "s2"),
			expectedServices: makeServices("s1", "s2"),
		},
		{
			name:             "single-name filter",
			filters:          []string{"s1"},
			services:         makeServices("s1", "s2"),
			expectedServices: makeServices("s1"),
		},
		{
			name:             "filter by prefix",
			filters:          []string{"prefix"},
			services:         makeServices("prefix-s1", "prefix-s2", "s2"),
			expectedServices: makeServices("prefix-s1", "prefix-s2"),
		},
		{
			name:             "multi-name filter",
			filters:          []string{"s1", "s2"},
			services:         makeServices("s1", "s2", "s3"),
			expectedServices: makeServices("s1", "s2"),
		},
		{
			name:             "stack name prefix is valid",
			filters:          []string{"stack_s1"},
			services:         makeServices("s1", "s11", "s2"),
			expectedServices: makeServices("s1", "s11"),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result := filterServicesByName(c.services, c.filters, "stack")
			assert.DeepEqual(t, c.expectedServices, result)
		})
	}
}

func makeServices(names ...string) []swarm.Service {
	result := make([]swarm.Service, len(names))
	for i, n := range names {
		result[i] = swarm.Service{Spec: swarm.ServiceSpec{Annotations: swarm.Annotations{Name: "stack_" + n}}}
	}
	return result
}
