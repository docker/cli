package system

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/golden"
)

func TestVersionWithoutServer(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		serverVersion: func(ctx context.Context, options client.ServerVersionOptions) (client.ServerVersionResult, error) {
			return client.ServerVersionResult{}, errors.New("no server")
		},
	})
	cmd := newVersionCommand(cli)
	cmd.SetArgs([]string{})
	cmd.SetOut(cli.Err())
	cmd.SetErr(io.Discard)
	assert.ErrorContains(t, cmd.Execute(), "no server")
	out := cli.OutBuffer().String()
	// TODO: use an assertion like e2e/image/build_test.go:assertBuildOutput()
	// instead of contains/not contains
	assert.Check(t, is.Contains(out, "Client:"))
	assert.Assert(t, !strings.Contains(out, "Server:"), "actual: %s", out)
}

func TestVersionFormat(t *testing.T) {
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
			Context:           "my-context",
		},
		Server: &serverVersion{
			Platform:      client.PlatformInfo{Name: "Docker Enterprise Edition (EE) 2.0"},
			Version:       "18.99.5-ce",
			APIVersion:    "1.30",
			MinAPIVersion: "1.12",
			Os:            "linux",
			Arch:          "amd64",
			GitCommit:     "64ddfa6",
			GoVersion:     "go1.8.7",
			KernelVersion: "v1.0.0",
			Experimental:  false,
			BuildTime:     "2018-07-09T22:38:38.000000000+00:00",
			Components: []system.ComponentVersion{
				{
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
				},
				{
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
				},
				{
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
				},
				{
					Name:    "Calico",
					Version: "v3.0.8",
					Details: map[string]string{
						"cni":              "v2.0.6",
						"kube-controllers": "v2.0.5",
						"node":             "v3.0.8",
					},
				},
			},
		},
	}

	tests := []struct {
		name   string
		format string
	}{
		{
			name: "default",
		},
		{
			name:   "json",
			format: "json",
		},
		{
			name:   "json template",
			format: "json",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpl, err := newVersionTemplate(tc.format)
			assert.NilError(t, err)

			var out bytes.Buffer
			assert.NilError(t, prettyPrintVersion(&out, vi, tmpl))
			assert.Check(t, golden.String(out.String(), t.Name()+".golden"))
		})
	}
}
