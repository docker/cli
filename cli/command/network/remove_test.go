package network

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

type forBiddenErr struct{ error }

func (forBiddenErr) Forbidden() {}

type notFoundErr struct{ error }

func (notFoundErr) NotFound() {}

func TestNetworkRemoveForce(t *testing.T) {
	tests := []struct {
		doc         string
		args        []string
		expectedErr string
	}{
		{
			doc:  "existing network",
			args: []string{"existing-network"},
		},
		{
			doc:  "existing network (forced)",
			args: []string{"--force", "existing-network"},
		},
		{
			doc:         "non-existing network",
			args:        []string{"no-such-network"},
			expectedErr: "no such network: no-such-network",
		},
		{
			doc:  "non-existing network (forced)",
			args: []string{"--force", "no-such-network"},
		},
		{
			doc:         "in-use network",
			args:        []string{"in-use-network"},
			expectedErr: "network is in use",
		},
		{
			doc:         "in-use network (forced)",
			args:        []string{"--force", "in-use-network"},
			expectedErr: "network is in use",
		},
		{
			doc:         "multiple networks",
			args:        []string{"existing-network", "no-such-network"},
			expectedErr: "no such network: no-such-network",
		},
		{
			doc:  "multiple networks (forced)",
			args: []string{"--force", "existing-network", "no-such-network"},
		},
		{
			doc:         "multiple networks 2 (forced)",
			args:        []string{"--force", "existing-network", "no-such-network", "in-use-network"},
			expectedErr: "network is in use",
		},
	}

	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			fakeCli := test.NewFakeCli(&fakeClient{
				networkRemoveFunc: func(ctx context.Context, networkID string) error {
					switch networkID {
					case "no-such-network":
						return notFoundErr{errors.New("no such network: no-such-network")}
					case "in-use-network":
						return forBiddenErr{errors.New("network is in use")}
					case "existing-network":
						return nil
					default:
						return nil
					}
				},
			})

			cmd := newRemoveCommand(fakeCli)
			cmd.SetOut(io.Discard)
			cmd.SetErr(fakeCli.ErrBuffer())
			cmd.SetArgs(tc.args)

			err := cmd.Execute()
			if tc.expectedErr == "" {
				assert.NilError(t, err)
			} else {
				assert.Check(t, is.Contains(fakeCli.ErrBuffer().String(), tc.expectedErr))
				assert.ErrorContains(t, err, "exit status 1")
			}
		})
	}
}

func TestNetworkRemovePromptTermination(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	cli := test.NewFakeCli(&fakeClient{
		networkRemoveFunc: func(ctx context.Context, networkID string) error {
			return errors.New("fakeClient networkRemoveFunc should not be called")
		},
		networkInspectFunc: func(ctx context.Context, networkID string, options client.NetworkInspectOptions) (client.NetworkInspectResult, error) {
			return client.NetworkInspectResult{
				Network: network.Inspect{
					Network: network.Network{
						ID:      "existing-network",
						Name:    "existing-network",
						Ingress: true,
					},
				},
			}, nil
		},
	})
	cmd := newRemoveCommand(cli)
	cmd.SetArgs([]string{"existing-network"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	test.TerminatePrompt(ctx, t, cmd, cli)
}
