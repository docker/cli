package image

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/docker/docker/pkg/streamformatter"
	"github.com/gotestyourself/gotestyourself/fs"
	"github.com/moby/buildkit/session"
	"github.com/stretchr/testify/require"
)

func TestAddDirToSession(t *testing.T) {
	dest := fs.NewDir(t, "test-build-session",
		fs.WithFile("Dockerfile", `
			FROM alpine:3.6
			COPY foo /
		`),
		fs.WithFile("foo", "some content", fs.AsUser(65534, 65534)),
	)
	defer dest.Remove()

	contextDir := dest.Path()
	sharedKey, err := getBuildSharedKey(contextDir)
	require.NoError(t, err)

	var s *session.Session
	s, err = session.NewSession(filepath.Base(contextDir), sharedKey)
	require.NoError(t, err)

	syncDone := make(chan error)
	progressOutput := streamformatter.NewProgressOutput(new(bytes.Buffer))
	err = addDirToSession(s, contextDir, progressOutput, syncDone)
	// Needs some assertions here to ensure we reset uid/gid to 0 for example
	require.NoError(t, err)
}
