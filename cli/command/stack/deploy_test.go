package stack

import (
	"bytes"
	"testing"

	"github.com/docker/cli/cli/compose/convert"
	"github.com/docker/cli/cli/internal/test"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestPruneServices(t *testing.T) {
	ctx := context.Background()
	namespace := convert.NewNamespace("foo")
	services := map[string]struct{}{
		"new":  {},
		"keep": {},
	}
	client := &fakeClient{services: []string{objectName("foo", "keep"), objectName("foo", "remove")}}
	dockerCli := test.NewFakeCliWithOutput(client, &bytes.Buffer{})
	dockerCli.SetErr(&bytes.Buffer{})

	pruneServices(ctx, dockerCli, namespace, services)

	assert.Equal(t, buildObjectIDs([]string{objectName("foo", "remove")}), client.removedServices)
}
