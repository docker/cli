package kubernetes

import (
	"testing"

	"github.com/docker/docker/api/types/filters"
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
			name:      "single-name filter",
			stackName: "test",
			filters:   filters.NewArgs(filters.KeyValuePair{Key: "name", Value: "svc-test"}),
			expectedSelectorParts: []string{
				"com.docker.stack.namespace=test",
				"com.docker.service.name=svc-test",
			},
		},
		{
			name:      "multi-name filter",
			stackName: "test",
			filters: filters.NewArgs(
				filters.KeyValuePair{Key: "name", Value: "svc-test"},
				filters.KeyValuePair{Key: "name", Value: "svc-test2"},
			),
			expectedSelectorParts: []string{
				"com.docker.stack.namespace=test",
				"com.docker.service.name in (svc-test,svc-test2)",
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
		{
			name:      "name filter with stackName prefix",
			stackName: "test",
			filters: filters.NewArgs(
				filters.KeyValuePair{Key: "name", Value: "test_svc1"},
			),
			expectedSelectorParts: []string{
				"com.docker.stack.namespace=test",
				"com.docker.service.name in (test_svc1,svc1)",
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
