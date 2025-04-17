package context

import (
	"testing"

	"github.com/docker/cli/cli/context/store"
	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

type fakeContextProvider struct {
	contextStore store.Store
}

func (c *fakeContextProvider) ContextStore() store.Store {
	return c.contextStore
}

func (*fakeContextProvider) CurrentContext() string {
	return "default"
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
	allNames := []string{"context-b", "context-c", "context-a"}
	cli := &fakeContextProvider{
		contextStore: fakeContextStore{
			names: allNames,
		},
	}

	t.Run("with limit", func(t *testing.T) {
		compFunc := completeContextNames(cli, 1, false)
		values, directives := compFunc(nil, nil, "")
		assert.Check(t, is.Equal(directives, cobra.ShellCompDirectiveNoFileComp))
		assert.Check(t, is.DeepEqual(values, allNames))

		values, directives = compFunc(nil, []string{"context-c"}, "")
		assert.Check(t, is.Equal(directives, cobra.ShellCompDirectiveNoFileComp))
		assert.Check(t, is.Len(values, 0))
	})

	t.Run("with limit and file completion", func(t *testing.T) {
		compFunc := completeContextNames(cli, 1, true)
		values, directives := compFunc(nil, nil, "")
		assert.Check(t, is.Equal(directives, cobra.ShellCompDirectiveNoFileComp))
		assert.Check(t, is.DeepEqual(values, allNames))

		values, directives = compFunc(nil, []string{"context-c"}, "")
		assert.Check(t, is.Equal(directives, cobra.ShellCompDirectiveDefault), "should provide filenames completion after limit")
		assert.Check(t, is.Len(values, 0))
	})

	t.Run("without limits", func(t *testing.T) {
		compFunc := completeContextNames(cli, -1, false)
		values, directives := compFunc(nil, []string{"context-c"}, "")
		assert.Check(t, is.Equal(directives, cobra.ShellCompDirectiveNoFileComp))
		assert.Check(t, is.DeepEqual(values, []string{"context-b", "context-a"}), "should not contain already completed")

		values, directives = compFunc(nil, []string{"context-c", "context-a"}, "")
		assert.Check(t, is.Equal(directives, cobra.ShellCompDirectiveNoFileComp))
		assert.Check(t, is.DeepEqual(values, []string{"context-b"}), "should not contain already completed")

		values, directives = compFunc(nil, []string{"context-c", "context-a", "context-b"}, "")
		assert.Check(t, is.Equal(directives, cobra.ShellCompDirectiveNoFileComp), "should provide filenames completion after limit")
		assert.Check(t, is.Len(values, 0))
	})
}
