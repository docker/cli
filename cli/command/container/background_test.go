package container

import (
	"os"
	"testing"

	"github.com/docker/cli/cli/streams"
	"gotest.tools/v3/assert"
)

func TestStdinForAttachNonTTY(t *testing.T) {
	r, w, err := os.Pipe()
	assert.NilError(t, err)
	t.Cleanup(func() {
		_ = r.Close()
		_ = w.Close()
	})

	in := streams.NewIn(r)
	got := stdinForAttach(in)
	assert.Equal(t, got, in)
}
