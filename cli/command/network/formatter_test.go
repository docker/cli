package network

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/stringid"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func TestNetworkContext(t *testing.T) {
	networkID := stringid.GenerateRandomID()

	var ctx networkContext
	cases := []struct {
		networkCtx networkContext
		expValue   string
		call       func() string
	}{
		{networkContext{
			n:     types.NetworkResource{ID: networkID},
			trunc: false,
		}, networkID, ctx.ID},
		{networkContext{
			n:     types.NetworkResource{ID: networkID},
			trunc: true,
		}, stringid.TruncateID(networkID), ctx.ID},
		{networkContext{
			n: types.NetworkResource{Name: "network_name"},
		}, "network_name", ctx.Name},
		{networkContext{
			n: types.NetworkResource{Driver: "driver_name"},
		}, "driver_name", ctx.Driver},
		{networkContext{
			n: types.NetworkResource{EnableIPv6: true},
		}, "true", ctx.IPv6},
		{networkContext{
			n: types.NetworkResource{EnableIPv6: false},
		}, "false", ctx.IPv6},
		{networkContext{
			n: types.NetworkResource{Internal: true},
		}, "true", ctx.Internal},
		{networkContext{
			n: types.NetworkResource{Internal: false},
		}, "false", ctx.Internal},
		{networkContext{
			n: types.NetworkResource{},
		}, "", ctx.Labels},
		{networkContext{
			n: types.NetworkResource{Labels: map[string]string{"label1": "value1", "label2": "value2"}},
		}, "label1=value1,label2=value2", ctx.Labels},
	}

	for _, c := range cases {
		ctx = c.networkCtx
		v := c.call()
		if strings.Contains(v, ",") {
			test.CompareMultipleValues(t, v, c.expValue)
		} else if v != c.expValue {
			t.Fatalf("Expected %s, was %s\n", c.expValue, v)
		}
	}
}

func TestNetworkContextWrite(t *testing.T) {
	cases := []struct {
		context  formatter.Context
		expected string
	}{

		// Errors
		{
			formatter.Context{Format: "{{InvalidFunction}}"},
			`Template parsing error: template: :1: function "InvalidFunction" not defined
`,
		},
		{
			formatter.Context{Format: "{{nil}}"},
			`Template parsing error: template: :1:2: executing "" at <nil>: nil is not a command
`,
		},
		// Table format
		{
			formatter.Context{Format: NewFormat("table", false)},
			`NETWORK ID          NAME                DRIVER              SCOPE
networkID1          foobar_baz          foo                 local
networkID2          foobar_bar          bar                 local
`,
		},
		{
			formatter.Context{Format: NewFormat("table", true)},
			`networkID1
networkID2
`,
		},
		{
			formatter.Context{Format: NewFormat("table {{.Name}}", false)},
			`NAME
foobar_baz
foobar_bar
`,
		},
		{
			formatter.Context{Format: NewFormat("table {{.Name}}", true)},
			`NAME
foobar_baz
foobar_bar
`,
		},
		// Raw Format
		{
			formatter.Context{Format: NewFormat("raw", false)},
			`network_id: networkID1
name: foobar_baz
driver: foo
scope: local

network_id: networkID2
name: foobar_bar
driver: bar
scope: local

`,
		},
		{
			formatter.Context{Format: NewFormat("raw", true)},
			`network_id: networkID1
network_id: networkID2
`,
		},
		// Custom Format
		{
			formatter.Context{Format: NewFormat("{{.Name}}", false)},
			`foobar_baz
foobar_bar
`,
		},
		// Custom Format with CreatedAt
		{
			formatter.Context{Format: NewFormat("{{.Name}} {{.CreatedAt}}", false)},
			`foobar_baz 2016-01-01 00:00:00 +0000 UTC
foobar_bar 2017-01-01 00:00:00 +0000 UTC
`,
		},
	}

	timestamp1, _ := time.Parse("2006-01-02", "2016-01-01")
	timestamp2, _ := time.Parse("2006-01-02", "2017-01-01")

	for _, testcase := range cases {
		networks := []types.NetworkResource{
			{ID: "networkID1", Name: "foobar_baz", Driver: "foo", Scope: "local", Created: timestamp1},
			{ID: "networkID2", Name: "foobar_bar", Driver: "bar", Scope: "local", Created: timestamp2},
		}
		out := bytes.NewBufferString("")
		testcase.context.Output = out
		err := FormatWrite(testcase.context, networks)
		if err != nil {
			assert.Error(t, err, testcase.expected)
		} else {
			assert.Check(t, is.Equal(testcase.expected, out.String()))
		}
	}
}

func TestNetworkContextWriteJSON(t *testing.T) {
	networks := []types.NetworkResource{
		{ID: "networkID1", Name: "foobar_baz"},
		{ID: "networkID2", Name: "foobar_bar"},
	}
	expectedJSONs := []map[string]interface{}{
		{"Driver": "", "ID": "networkID1", "IPv6": "false", "Internal": "false", "Labels": "", "Name": "foobar_baz", "Scope": "", "CreatedAt": "0001-01-01 00:00:00 +0000 UTC"},
		{"Driver": "", "ID": "networkID2", "IPv6": "false", "Internal": "false", "Labels": "", "Name": "foobar_bar", "Scope": "", "CreatedAt": "0001-01-01 00:00:00 +0000 UTC"},
	}

	out := bytes.NewBufferString("")
	err := FormatWrite(formatter.Context{Format: "{{json .}}", Output: out}, networks)
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		msg := fmt.Sprintf("Output: line %d: %s", i, line)
		var m map[string]interface{}
		err := json.Unmarshal([]byte(line), &m)
		assert.NilError(t, err, msg)
		assert.Check(t, is.DeepEqual(expectedJSONs[i], m), msg)
	}
}

func TestNetworkContextWriteJSONField(t *testing.T) {
	networks := []types.NetworkResource{
		{ID: "networkID1", Name: "foobar_baz"},
		{ID: "networkID2", Name: "foobar_bar"},
	}
	out := bytes.NewBufferString("")
	err := FormatWrite(formatter.Context{Format: "{{json .ID}}", Output: out}, networks)
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		msg := fmt.Sprintf("Output: line %d: %s", i, line)
		var s string
		err := json.Unmarshal([]byte(line), &s)
		assert.NilError(t, err, msg)
		assert.Check(t, is.Equal(networks[i].ID, s), msg)
	}
}
