package main

import (
	"testing"

	"github.com/docker/cli/cli/context/store"
	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

type fakeCLI struct {
	contextStore store.Store
}

func (c *fakeCLI) ContextStore() store.Store {
	return c.contextStore
}

type fakeContextStore struct {
	store.Store
	names []string
}

func (f fakeContextStore) List() (c []store.Metadata, _ error) {
	for _, name := range f.names {
		c = append(c, store.Metadata{Name: name})
	}
	return c, nil
}

func TestCompleteContextNames(t *testing.T) {
	expectedNames := []string{"context-b", "context-c", "context-a"}
	cli := &fakeCLI{
		contextStore: fakeContextStore{
			names: expectedNames,
		},
	}

	values, directives := completeContextNames(cli)(nil, nil, "")
	assert.Check(t, is.Equal(directives, cobra.ShellCompDirectiveNoFileComp))
	assert.Check(t, is.DeepEqual(values, expectedNames))
}

func TestCompleteLogLevels(t *testing.T) {
	values, directives := completeLogLevels(nil, nil, "")
	assert.Check(t, is.Equal(directives, cobra.ShellCompDirectiveNoFileComp))
	assert.Check(t, is.DeepEqual(values, logLevels))
}
