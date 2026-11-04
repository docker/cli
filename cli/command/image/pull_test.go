package image

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/containerd/errdefs"
	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/api/types/registry"
	"github.com/moby/moby/client"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/golden"
)

func TestNewPullCommandErrors(t *testing.T) {
	testCases := []struct {
		name          string
		args          []string
		expectedError string
	}{
		{
			name:          "wrong-args",
			expectedError: "requires 1 argument",
			args:          []string{},
		},
		{
			name:          "invalid-name",
			expectedError: "invalid reference format: repository name (library/UPPERCASE_REPO) must be lowercase",
			args:          []string{"UPPERCASE_REPO"},
		},
		{
			name:          "all-tags-with-tag",
			expectedError: "tag can't be used with --all-tags/-a",
			args:          []string{"--all-tags", "image:tag"},
		},
		{
			name:          "all-tags-with-dry-run",
			expectedError: "if any flags in the group [all-tags dry-run] are set none of the others can be",
			args:          []string{"--all-tags", "--dry-run", "image"},
		},
		{
			name:          "format-without-dry-run",
			expectedError: "--format can only be used with --dry-run",
			args:          []string{"--format", "json", "image"},
		},
		{
			name:          "dry-run-invalid-template",
			expectedError: "template parsing error",
			args:          []string{"--dry-run", "--format", "{{.Status", "image"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{})
			cmd := newPullCommand(cli)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(tc.args)
			assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
		})
	}
}

func TestNewPullCommandSuccess(t *testing.T) {
	testCases := []struct {
		name        string
		args        []string
		expectedTag string
	}{
		{
			name:        "simple",
			args:        []string{"image:tag"},
			expectedTag: "image:tag",
		},
		{
			name:        "simple-no-tag",
			args:        []string{"image"},
			expectedTag: "image:latest",
		},
		{
			name:        "simple-quiet",
			args:        []string{"--quiet", "image"},
			expectedTag: "image:latest",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{
				imagePullFunc: func(ref string, options client.ImagePullOptions) (client.ImagePullResponse, error) {
					assert.Check(t, is.Equal(tc.expectedTag, ref), tc.name)
					// FIXME(thaJeztah): how to mock this?
					return fakeStreamResult{ReadCloser: http.NoBody}, nil
				},
			})
			cmd := newPullCommand(cli)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			assert.NilError(t, err)
			golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("pull-command-success.%s.golden", tc.name))
		})
	}
}

func TestNewPullCommandDryRun(t *testing.T) {
	const remoteDigest = "sha256:c0537ff6a5218ef531ece93d4984efc99bbf3f7497c0a7726c88e2bb7584dc96"

	testCases := []struct {
		name        string
		args        []string
		repoDigests []string
		localError  error
	}{
		{
			name:        "up-to-date",
			args:        []string{"--dry-run", "image:tag"},
			repoDigests: []string{"docker.io/library/image@" + remoteDigest},
		},
		{
			name:        "newer-available",
			args:        []string{"--dry-run", "image:tag"},
			repoDigests: []string{"docker.io/library/image@sha256:1b25a9b06ee76ff3d21a34b33bd8a1a9e4ef4e2b2a3f4f43d4db1f81a1e4e00a"},
		},
		{
			name:       "not-present-locally",
			args:       []string{"--dry-run", "image:tag"},
			localError: errdefs.ErrNotFound.WithMessage("No such image: image:tag"),
		},
		{
			name:        "format-json",
			args:        []string{"--dry-run", "--format", "json", "image:tag"},
			repoDigests: []string{"docker.io/library/image@" + remoteDigest},
		},
		{
			name:        "format-template",
			args:        []string{"--dry-run", "--format", "{{.Status}}", "image:tag"},
			repoDigests: []string{"docker.io/library/image@sha256:1b25a9b06ee76ff3d21a34b33bd8a1a9e4ef4e2b2a3f4f43d4db1f81a1e4e00a"},
		},
		{
			name:        "quiet",
			args:        []string{"--dry-run", "--quiet", "image:tag"},
			repoDigests: []string{"docker.io/library/image@" + remoteDigest},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{
				imagePullFunc: func(ref string, options client.ImagePullOptions) (client.ImagePullResponse, error) {
					return fakeStreamResult{}, errors.New("ImagePull should not be called with --dry-run")
				},
				distributionInspectFunc: func(image string, options client.DistributionInspectOptions) (client.DistributionInspectResult, error) {
					assert.Check(t, is.Equal("image:tag", image))
					return client.DistributionInspectResult{
						DistributionInspect: registry.DistributionInspect{
							Descriptor: ocispec.Descriptor{Digest: remoteDigest},
						},
					}, nil
				},
				imageInspectFunc: func(img string) (client.ImageInspectResult, error) {
					if tc.localError != nil {
						return client.ImageInspectResult{}, tc.localError
					}
					return client.ImageInspectResult{
						InspectResponse: image.InspectResponse{RepoDigests: tc.repoDigests},
					}, nil
				},
			})
			cmd := newPullCommand(cli)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			assert.NilError(t, err)
			golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("pull-command-dry-run.%s.golden", tc.name))
		})
	}
}
