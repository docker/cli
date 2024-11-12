// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.22

package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/pkg/stringid"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestNodeContext(t *testing.T) {
	nodeID := stringid.GenerateRandomID()

	var ctx nodeContext
	cases := []struct {
		nodeCtx  nodeContext
		expValue string
		call     func() string
	}{
		{nodeContext{
			n: swarm.Node{ID: nodeID},
		}, nodeID, ctx.ID},
		{nodeContext{
			n: swarm.Node{Description: swarm.NodeDescription{Hostname: "node_hostname"}},
		}, "node_hostname", ctx.Hostname},
		{nodeContext{
			n: swarm.Node{Status: swarm.NodeStatus{State: swarm.NodeState("foo")}},
		}, "Foo", ctx.Status},
		{nodeContext{
			n: swarm.Node{Spec: swarm.NodeSpec{Availability: swarm.NodeAvailability("drain")}},
		}, "Drain", ctx.Availability},
		{nodeContext{
			n: swarm.Node{ManagerStatus: &swarm.ManagerStatus{Leader: true}},
		}, "Leader", ctx.ManagerStatus},
	}

	for _, c := range cases {
		ctx = c.nodeCtx
		v := c.call()
		if strings.Contains(v, ",") {
			test.CompareMultipleValues(t, v, c.expValue)
		} else if v != c.expValue {
			t.Fatalf("Expected %s, was %s\n", c.expValue, v)
		}
	}
}

func TestNodeContextWrite(t *testing.T) {
	cases := []struct {
		context     formatter.Context
		expected    string
		clusterInfo swarm.ClusterInfo
	}{
		// Errors
		{
			context:     formatter.Context{Format: "{{InvalidFunction}}"},
			expected:    `template parsing error: template: :1: function "InvalidFunction" not defined`,
			clusterInfo: swarm.ClusterInfo{TLSInfo: swarm.TLSInfo{TrustRoot: "hi"}},
		},
		{
			context:     formatter.Context{Format: "{{nil}}"},
			expected:    `template parsing error: template: :1:2: executing "" at <nil>: nil is not a command`,
			clusterInfo: swarm.ClusterInfo{TLSInfo: swarm.TLSInfo{TrustRoot: "hi"}},
		},
		// Table format
		{
			context: formatter.Context{Format: NewFormat("table", false)},
			expected: `ID          HOSTNAME     STATUS    AVAILABILITY   MANAGER STATUS   ENGINE VERSION
nodeID1     foobar_baz   Foo       Drain          Leader           18.03.0-ce
nodeID2     foobar_bar   Bar       Active         Reachable        1.2.3
nodeID3     foobar_boo   Boo       Active                          ` + "\n", // (to preserve whitespace)
			clusterInfo: swarm.ClusterInfo{TLSInfo: swarm.TLSInfo{TrustRoot: "hi"}},
		},
		{
			context: formatter.Context{Format: NewFormat("table", true)},
			expected: `nodeID1
nodeID2
nodeID3
`,
			clusterInfo: swarm.ClusterInfo{TLSInfo: swarm.TLSInfo{TrustRoot: "hi"}},
		},
		{
			context: formatter.Context{Format: NewFormat("table {{.Hostname}}", false)},
			expected: `HOSTNAME
foobar_baz
foobar_bar
foobar_boo
`,
			clusterInfo: swarm.ClusterInfo{TLSInfo: swarm.TLSInfo{TrustRoot: "hi"}},
		},
		{
			context: formatter.Context{Format: NewFormat("table {{.Hostname}}", true)},
			expected: `HOSTNAME
foobar_baz
foobar_bar
foobar_boo
`,
			clusterInfo: swarm.ClusterInfo{TLSInfo: swarm.TLSInfo{TrustRoot: "hi"}},
		},
		{
			context: formatter.Context{Format: NewFormat("table {{.ID}}\t{{.Hostname}}\t{{.TLSStatus}}", false)},
			expected: `ID        HOSTNAME     TLS STATUS
nodeID1   foobar_baz   Needs Rotation
nodeID2   foobar_bar   Ready
nodeID3   foobar_boo   Unknown
`,
			clusterInfo: swarm.ClusterInfo{TLSInfo: swarm.TLSInfo{TrustRoot: "hi"}},
		},
		{ // no cluster TLS status info, TLS status for all nodes is unknown
			context: formatter.Context{Format: NewFormat("table {{.ID}}\t{{.Hostname}}\t{{.TLSStatus}}", false)},
			expected: `ID        HOSTNAME     TLS STATUS
nodeID1   foobar_baz   Unknown
nodeID2   foobar_bar   Unknown
nodeID3   foobar_boo   Unknown
`,
			clusterInfo: swarm.ClusterInfo{},
		},
		// Raw Format
		{
			context: formatter.Context{Format: NewFormat("raw", false)},
			expected: `node_id: nodeID1
hostname: foobar_baz
status: Foo
availability: Drain
manager_status: Leader

node_id: nodeID2
hostname: foobar_bar
status: Bar
availability: Active
manager_status: Reachable

node_id: nodeID3
hostname: foobar_boo
status: Boo
availability: Active
manager_status: ` + "\n\n", // to preserve whitespace
			clusterInfo: swarm.ClusterInfo{TLSInfo: swarm.TLSInfo{TrustRoot: "hi"}},
		},
		{
			context: formatter.Context{Format: NewFormat("raw", true)},
			expected: `node_id: nodeID1
node_id: nodeID2
node_id: nodeID3
`,
			clusterInfo: swarm.ClusterInfo{TLSInfo: swarm.TLSInfo{TrustRoot: "hi"}},
		},
		// Custom Format
		{
			context: formatter.Context{Format: NewFormat("{{.Hostname}}  {{.TLSStatus}}", false)},
			expected: `foobar_baz  Needs Rotation
foobar_bar  Ready
foobar_boo  Unknown
`,
			clusterInfo: swarm.ClusterInfo{TLSInfo: swarm.TLSInfo{TrustRoot: "hi"}},
		},
	}

	nodes := []swarm.Node{
		{
			ID: "nodeID1",
			Description: swarm.NodeDescription{
				Hostname: "foobar_baz",
				TLSInfo:  swarm.TLSInfo{TrustRoot: "no"},
				Engine:   swarm.EngineDescription{EngineVersion: "18.03.0-ce"},
			},
			Status:        swarm.NodeStatus{State: swarm.NodeState("foo")},
			Spec:          swarm.NodeSpec{Availability: swarm.NodeAvailability("drain")},
			ManagerStatus: &swarm.ManagerStatus{Leader: true},
		},
		{
			ID: "nodeID2",
			Description: swarm.NodeDescription{
				Hostname: "foobar_bar",
				TLSInfo:  swarm.TLSInfo{TrustRoot: "hi"},
				Engine:   swarm.EngineDescription{EngineVersion: "1.2.3"},
			},
			Status: swarm.NodeStatus{State: swarm.NodeState("bar")},
			Spec:   swarm.NodeSpec{Availability: swarm.NodeAvailability("active")},
			ManagerStatus: &swarm.ManagerStatus{
				Leader:       false,
				Reachability: swarm.Reachability("Reachable"),
			},
		},
		{
			ID:          "nodeID3",
			Description: swarm.NodeDescription{Hostname: "foobar_boo"},
			Status:      swarm.NodeStatus{State: swarm.NodeState("boo")},
			Spec:        swarm.NodeSpec{Availability: swarm.NodeAvailability("active")},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(string(tc.context.Format), func(t *testing.T) {
			var out bytes.Buffer
			tc.context.Output = &out

			err := FormatWrite(tc.context, nodes, system.Info{Swarm: swarm.Info{Cluster: &tc.clusterInfo}})
			if err != nil {
				assert.Error(t, err, tc.expected)
			} else {
				assert.Equal(t, out.String(), tc.expected)
			}
		})
	}
}

func TestNodeContextWriteJSON(t *testing.T) {
	cases := []struct {
		expected []map[string]any
		info     system.Info
	}{
		{
			expected: []map[string]any{
				{"Availability": "", "Hostname": "foobar_baz", "ID": "nodeID1", "ManagerStatus": "", "Status": "", "Self": false, "TLSStatus": "Unknown", "EngineVersion": "1.2.3"},
				{"Availability": "", "Hostname": "foobar_bar", "ID": "nodeID2", "ManagerStatus": "", "Status": "", "Self": false, "TLSStatus": "Unknown", "EngineVersion": ""},
				{"Availability": "", "Hostname": "foobar_boo", "ID": "nodeID3", "ManagerStatus": "", "Status": "", "Self": false, "TLSStatus": "Unknown", "EngineVersion": "18.03.0-ce"},
			},
			info: system.Info{},
		},
		{
			expected: []map[string]any{
				{"Availability": "", "Hostname": "foobar_baz", "ID": "nodeID1", "ManagerStatus": "", "Status": "", "Self": false, "TLSStatus": "Ready", "EngineVersion": "1.2.3"},
				{"Availability": "", "Hostname": "foobar_bar", "ID": "nodeID2", "ManagerStatus": "", "Status": "", "Self": false, "TLSStatus": "Needs Rotation", "EngineVersion": ""},
				{"Availability": "", "Hostname": "foobar_boo", "ID": "nodeID3", "ManagerStatus": "", "Status": "", "Self": false, "TLSStatus": "Unknown", "EngineVersion": "18.03.0-ce"},
			},
			info: system.Info{
				Swarm: swarm.Info{
					Cluster: &swarm.ClusterInfo{
						TLSInfo:                swarm.TLSInfo{TrustRoot: "hi"},
						RootRotationInProgress: true,
					},
				},
			},
		},
	}

	for _, testcase := range cases {
		nodes := []swarm.Node{
			{ID: "nodeID1", Description: swarm.NodeDescription{Hostname: "foobar_baz", TLSInfo: swarm.TLSInfo{TrustRoot: "hi"}, Engine: swarm.EngineDescription{EngineVersion: "1.2.3"}}},
			{ID: "nodeID2", Description: swarm.NodeDescription{Hostname: "foobar_bar", TLSInfo: swarm.TLSInfo{TrustRoot: "no"}}},
			{ID: "nodeID3", Description: swarm.NodeDescription{Hostname: "foobar_boo", Engine: swarm.EngineDescription{EngineVersion: "18.03.0-ce"}}},
		}
		out := bytes.NewBufferString("")
		err := FormatWrite(formatter.Context{Format: "{{json .}}", Output: out}, nodes, testcase.info)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
			msg := fmt.Sprintf("Output: line %d: %s", i, line)
			var m map[string]any
			err := json.Unmarshal([]byte(line), &m)
			assert.NilError(t, err, msg)
			assert.Check(t, is.DeepEqual(testcase.expected[i], m), msg)
		}
	}
}

func TestNodeContextWriteJSONField(t *testing.T) {
	nodes := []swarm.Node{
		{ID: "nodeID1", Description: swarm.NodeDescription{Hostname: "foobar_baz"}},
		{ID: "nodeID2", Description: swarm.NodeDescription{Hostname: "foobar_bar"}},
	}
	out := bytes.NewBufferString("")
	err := FormatWrite(formatter.Context{Format: "{{json .ID}}", Output: out}, nodes, system.Info{})
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		msg := fmt.Sprintf("Output: line %d: %s", i, line)
		var s string
		err := json.Unmarshal([]byte(line), &s)
		assert.NilError(t, err, msg)
		assert.Check(t, is.Equal(nodes[i].ID, s), msg)
	}
}

func TestNodeInspectWriteContext(t *testing.T) {
	node := swarm.Node{
		ID: "nodeID1",
		Description: swarm.NodeDescription{
			Hostname: "foobar_baz",
			TLSInfo: swarm.TLSInfo{
				TrustRoot:           "-----BEGIN CERTIFICATE-----\ndata\n-----END CERTIFICATE-----\n",
				CertIssuerPublicKey: []byte("pubKey"),
				CertIssuerSubject:   []byte("subject"),
			},
			Platform: swarm.Platform{
				OS:           "linux",
				Architecture: "amd64",
			},
			Resources: swarm.Resources{
				MemoryBytes: 1,
			},
			Engine: swarm.EngineDescription{
				EngineVersion: "0.1.1",
			},
		},
		Status: swarm.NodeStatus{
			State: swarm.NodeState("ready"),
			Addr:  "1.1.1.1",
		},
		Spec: swarm.NodeSpec{
			Availability: swarm.NodeAvailability("drain"),
			Role:         swarm.NodeRole("manager"),
		},
	}
	out := bytes.NewBufferString("")
	context := formatter.Context{
		Format: NewFormat("pretty", false),
		Output: out,
	}
	err := InspectFormatWrite(context, []string{"nodeID1"}, func(string) (any, []byte, error) {
		return node, nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := `ID:			nodeID1
Hostname:              	foobar_baz
Joined at:             	0001-01-01 00:00:00 +0000 utc
Status:
 State:			Ready
 Availability:         	Drain
 Address:		1.1.1.1
Platform:
 Operating System:	linux
 Architecture:		amd64
Resources:
 CPUs:			0
 Memory:		1B
Engine Version:		0.1.1
TLS Info:
 TrustRoot:
-----BEGIN CERTIFICATE-----
data
-----END CERTIFICATE-----

 Issuer Subject:	c3ViamVjdA==
 Issuer Public Key:	cHViS2V5
`
	assert.Check(t, is.Equal(expected, out.String()))
}
