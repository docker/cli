package node

import (
	"io/ioutil"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/gotestyourself/gotestyourself/golden"
	"github.com/pkg/errors"
	// Import builders to get the builder function as package function
	. "github.com/docker/cli/internal/test/builders"
)

func TestNodeListErrorOnAPIFailure(t *testing.T) {
	testCases := []struct {
		nodeListFunc  func() ([]swarm.Node, error)
		infoFunc      func() (types.Info, error)
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
			infoFunc: func() (types.Info, error) {
				return types.Info{}, errors.Errorf("error asking for node info")
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
		cmd.SetOutput(ioutil.Discard)
		assert.Error(t, cmd.Execute(), tc.expectedError)
	}
}

func TestNodeList(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		nodeListFunc: func() ([]swarm.Node, error) {
			return []swarm.Node{
				*Node(NodeID("nodeID1"), Hostname("node-2-foo"), Manager(Leader()), EngineVersion(".")),
				*Node(NodeID("nodeID2"), Hostname("node-10-foo"), Manager(), EngineVersion("18.03.0-ce")),
				*Node(NodeID("nodeID3"), Hostname("node-1-foo")),
			}, nil
		},
		infoFunc: func() (types.Info, error) {
			return types.Info{
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
				*Node(NodeID("nodeID1")),
			}, nil
		},
	})
	cmd := newListCommand(cli)
	cmd.Flags().Set("quiet", "true")
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Equal(cli.OutBuffer().String(), "nodeID1\n"))
}

func TestNodeListDefaultFormatFromConfig(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		nodeListFunc: func() ([]swarm.Node, error) {
			return []swarm.Node{
				*Node(NodeID("nodeID1"), Hostname("nodeHostname1"), Manager(Leader())),
				*Node(NodeID("nodeID2"), Hostname("nodeHostname2"), Manager()),
				*Node(NodeID("nodeID3"), Hostname("nodeHostname3")),
			}, nil
		},
		infoFunc: func() (types.Info, error) {
			return types.Info{
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
				*Node(NodeID("nodeID1"), Hostname("nodeHostname1"), Manager(Leader())),
				*Node(NodeID("nodeID2"), Hostname("nodeHostname2"), Manager()),
			}, nil
		},
		infoFunc: func() (types.Info, error) {
			return types.Info{
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
	cmd.Flags().Set("format", "{{.Hostname}}: {{.ManagerStatus}}")
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "node-list-format-flag.golden")
}
