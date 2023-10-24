package node

import (
	"io"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/builders"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/system"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/golden"
)

func TestNodeListErrorOnAPIFailure(t *testing.T) {
	testCases := []struct {
		nodeListFunc  func() ([]swarm.Node, error)
		infoFunc      func() (system.Info, error)
		expectedError string
	}{
		{
			nodeListFunc: func() ([]swarm.Node, error) {
				return []swarm.Node{}, errors.Errorf("error listing nodes")
			},
			expectedError: "error listing nodes",
		},
		{
			nodeListFunc: func() ([]swarm.Node, error) {
				return []swarm.Node{
					{
						ID: "nodeID",
					},
				}, nil
			},
			infoFunc: func() (system.Info, error) {
				return system.Info{}, errors.Errorf("error asking for node info")
			},
			expectedError: "error asking for node info",
		},
	}
	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{
			nodeListFunc: tc.nodeListFunc,
			infoFunc:     tc.infoFunc,
		})
		cmd := newListCommand(cli)
		cmd.SetOut(io.Discard)
		assert.Error(t, cmd.Execute(), tc.expectedError)
	}
}

func TestNodeList(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		nodeListFunc: func() ([]swarm.Node, error) {
			return []swarm.Node{
				*builders.Node(builders.NodeID("nodeID1"), builders.Hostname("node-2-foo"), builders.Manager(builders.Leader()), builders.EngineVersion(".")),
				*builders.Node(builders.NodeID("nodeID2"), builders.Hostname("node-10-foo"), builders.Manager(), builders.EngineVersion("18.03.0-ce")),
				*builders.Node(builders.NodeID("nodeID3"), builders.Hostname("node-1-foo")),
			}, nil
		},
		infoFunc: func() (system.Info, error) {
			return system.Info{
				Swarm: swarm.Info{
					NodeID: "nodeID1",
				},
			}, nil
		},
	})

	cmd := newListCommand(cli)
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "node-list-sort.golden")
}

func TestNodeListQuietShouldOnlyPrintIDs(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		nodeListFunc: func() ([]swarm.Node, error) {
			return []swarm.Node{
				*builders.Node(builders.NodeID("nodeID1")),
			}, nil
		},
	})
	cmd := newListCommand(cli)
	assert.Check(t, cmd.Flags().Set("quiet", "true"))
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Equal(cli.OutBuffer().String(), "nodeID1\n"))
}

func TestNodeListDefaultFormatFromConfig(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		nodeListFunc: func() ([]swarm.Node, error) {
			return []swarm.Node{
				*builders.Node(builders.NodeID("nodeID1"), builders.Hostname("nodeHostname1"), builders.Manager(builders.Leader())),
				*builders.Node(builders.NodeID("nodeID2"), builders.Hostname("nodeHostname2"), builders.Manager()),
				*builders.Node(builders.NodeID("nodeID3"), builders.Hostname("nodeHostname3")),
			}, nil
		},
		infoFunc: func() (system.Info, error) {
			return system.Info{
				Swarm: swarm.Info{
					NodeID: "nodeID1",
				},
			}, nil
		},
	})
	cli.SetConfigFile(&configfile.ConfigFile{
		NodesFormat: "{{.ID}}: {{.Hostname}} {{.Status}}/{{.ManagerStatus}}",
	})
	cmd := newListCommand(cli)
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "node-list-format-from-config.golden")
}

func TestNodeListFormat(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		nodeListFunc: func() ([]swarm.Node, error) {
			return []swarm.Node{
				*builders.Node(builders.NodeID("nodeID1"), builders.Hostname("nodeHostname1"), builders.Manager(builders.Leader())),
				*builders.Node(builders.NodeID("nodeID2"), builders.Hostname("nodeHostname2"), builders.Manager()),
			}, nil
		},
		infoFunc: func() (system.Info, error) {
			return system.Info{
				Swarm: swarm.Info{
					NodeID: "nodeID1",
				},
			}, nil
		},
	})
	cli.SetConfigFile(&configfile.ConfigFile{
		NodesFormat: "{{.ID}}: {{.Hostname}} {{.Status}}/{{.ManagerStatus}}",
	})
	cmd := newListCommand(cli)
	assert.Check(t, cmd.Flags().Set("format", "{{.Hostname}}: {{.ManagerStatus}}"))
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "node-list-format-flag.golden")
}
