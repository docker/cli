package system

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"gotest.tools/golden"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api"
	"github.com/docker/docker/api/types"
)

func TestVersionWithoutServer(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		serverVersion: func(ctx context.Context) (types.Version, error) {
			return types.Version{}, fmt.Errorf("no server")
		},
	})
	cmd := NewVersionCommand(cli)
	cmd.SetOutput(cli.Err())
	assert.ErrorContains(t, cmd.Execute(), "no server")
	out := cli.OutBuffer().String()
	// TODO: use an assertion like e2e/image/build_test.go:assertBuildOutput()
	// instead of contains/not contains
	assert.Check(t, is.Contains(out, "Client:"))
	assert.Assert(t, !strings.Contains(out, "Server:"), "actual: %s", out)
}

func fakeServerVersion(_ context.Context) (types.Version, error) {
	return types.Version{
		Version:    "docker-dev",
		APIVersion: api.DefaultVersion,
	}, nil
}

func TestVersionWithOrchestrator(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{serverVersion: fakeServerVersion})
	cmd := NewVersionCommand(cli)
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Contains(cleanTabs(cli.OutBuffer().String()), "Orchestrator: swarm"))
}

func TestVersionAlign(t *testing.T) {
	vi := versionInfo{
		Client: clientVersion{
			Version:           "18.99.5-ce",
			APIVersion:        "1.38",
			DefaultAPIVersion: "1.38",
			GitCommit:         "deadbeef",
			GoVersion:         "go1.10.2",
			Os:                "linux",
			Arch:              "amd64",
			BuildTime:         "Wed May 30 22:21:05 2018",
			Experimental:      true,
			StackOrchestrator: "swarm",
		},
	}

	cli := test.NewFakeCli(&fakeClient{})
	tmpl, err := newVersionTemplate("")
	assert.NilError(t, err)
	assert.NilError(t, prettyPrintVersion(cli, vi, tmpl))
	assert.Check(t, golden.String(cli.OutBuffer().String(), "docker-client-version.golden"))
	assert.Check(t, is.Equal("", cli.ErrBuffer().String()))
}

func cleanTabs(line string) string {
	return strings.Join(strings.Fields(line), " ")
}
