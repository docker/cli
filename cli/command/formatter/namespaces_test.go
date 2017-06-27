package formatter

import (
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/stringid"
)

func TestNamespacesPsContext(t *testing.T) {
	containerID := stringid.GenerateRandomID()

	var ctx namespacesContext

	cases := []struct {
		container types.Container
		trunc     bool
		expValue  string
		call      func() string
	}{
		{
			container: types.Container{
				ID: containerID,
			},
			trunc:    true,
			expValue: stringid.TruncateID(containerID),
			call:     ctx.ID,
		},
		{
			container: types.Container{
				ID: containerID,
			},
			trunc:    false,
			expValue: containerID,
			call:     ctx.ID,
		},
	}

	for _, c := range cases {
		ctx = namespacesContext{c: c.container, trunc: c.trunc}

		v := c.call()

		if strings.Contains(v, ",") {
			compareMultipleValues(t, v, c.expValue)
		} else if v != c.expValue {
			t.Fatalf("Expected %s, was %s\n", c.expValue, v)
		}
	}
}
