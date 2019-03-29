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
		},
		Server: &types.Version{},
	}

	vi.Server.Platform.Name = "Docker Enterprise Edition (EE) 2.0"

	vi.Server.Components = append(vi.Server.Components, types.ComponentVersion{
		Name:    "Engine",
		Version: "17.06.2-ee-15",
		Details: map[string]string{
			"ApiVersion":    "1.30",
			"MinAPIVersion": "1.12",
			"GitCommit":     "64ddfa6",
			"GoVersion":     "go1.8.7",
			"Os":            "linux",
			"Arch":          "amd64",
			"BuildTime":     "Mon Jul  9 23:38:38 2018",
			"Experimental":  "false",
		},
	})

	vi.Server.Components = append(vi.Server.Components, types.ComponentVersion{
		Name:    "Universal Control Plane",
		Version: "17.06.2-ee-15",
		Details: map[string]string{
			"Version":       "3.0.3-tp2",
			"ApiVersion":    "1.30",
			"Arch":          "amd64",
			"BuildTime":     "Mon Jul  2 21:24:07 UTC 2018",
			"GitCommit":     "4513922",
			"GoVersion":     "go1.9.4",
			"MinApiVersion": "1.20",
			"Os":            "linux",
		},
	})

	vi.Server.Components = append(vi.Server.Components, types.ComponentVersion{
		Name:    "Kubernetes",
		Version: "1.8+",
		Details: map[string]string{
			"buildDate":    "2018-04-26T16:51:21Z",
			"compiler":     "gc",
			"gitCommit":    "8d637aedf46b9c21dde723e29c645b9f27106fa5",
			"gitTreeState": "clean",
			"gitVersion":   "v1.8.11-docker-8d637ae",
			"goVersion":    "go1.8.3",
			"major":        "1",
			"minor":        "8+",
			"platform":     "linux/amd64",
		},
	})

	vi.Server.Components = append(vi.Server.Components, types.ComponentVersion{
		Name:    "Calico",
		Version: "v3.0.8",
		Details: map[string]string{
			"cni":              "v2.0.6",
			"kube-controllers": "v2.0.5",
			"node":             "v3.0.8",
		},
	})

	cli := test.NewFakeCli(&fakeClient{})
	tmpl, err := newVersionTemplate("")
	assert.NilError(t, err)
	assert.NilError(t, prettyPrintVersion(cli, vi, tmpl))
	assert.Check(t, golden.String(cli.OutBuffer().String(), "docker-client-version.golden"))
	assert.Check(t, is.Equal("", cli.ErrBuffer().String()))
}
