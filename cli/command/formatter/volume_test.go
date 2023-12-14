// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.19

package formatter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/pkg/stringid"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestVolumeContext(t *testing.T) {
	volumeName := stringid.GenerateRandomID()

	var ctx volumeContext
	cases := []struct {
		volumeCtx volumeContext
		expValue  string
		call      func() string
	}{
		{volumeContext{
			v: volume.Volume{Name: volumeName},
		}, volumeName, ctx.Name},
		{volumeContext{
			v: volume.Volume{Driver: "driver_name"},
		}, "driver_name", ctx.Driver},
		{volumeContext{
			v: volume.Volume{Scope: "local"},
		}, "local", ctx.Scope},
		{volumeContext{
			v: volume.Volume{Mountpoint: "mountpoint"},
		}, "mountpoint", ctx.Mountpoint},
		{volumeContext{
			v: volume.Volume{},
		}, "", ctx.Labels},
		{volumeContext{
			v: volume.Volume{Labels: map[string]string{"label1": "value1", "label2": "value2"}},
		}, "label1=value1,label2=value2", ctx.Labels},
	}

	for _, c := range cases {
		ctx = c.volumeCtx
		v := c.call()
		if strings.Contains(v, ",") {
			test.CompareMultipleValues(t, v, c.expValue)
		} else if v != c.expValue {
			t.Fatalf("Expected %s, was %s\n", c.expValue, v)
		}
	}
}

func TestVolumeContextWrite(t *testing.T) {
	cases := []struct {
		context  Context
		expected string
	}{
		// Errors
		{
			Context{Format: "{{InvalidFunction}}"},
			`template parsing error: template: :1: function "InvalidFunction" not defined`,
		},
		{
			Context{Format: "{{nil}}"},
			`template parsing error: template: :1:2: executing "" at <nil>: nil is not a command`,
		},
		// Table format
		{
			Context{Format: NewVolumeFormat("table", false)},
			`DRIVER    VOLUME NAME
foo       foobar_baz
bar       foobar_bar
`,
		},
		{
			Context{Format: NewVolumeFormat("table", true)},
			`foobar_baz
foobar_bar
`,
		},
		{
			Context{Format: NewVolumeFormat("table {{.Name}}", false)},
			`VOLUME NAME
foobar_baz
foobar_bar
`,
		},
		{
			Context{Format: NewVolumeFormat("table {{.Name}}", true)},
			`VOLUME NAME
foobar_baz
foobar_bar
`,
		},
		// Raw Format
		{
			Context{Format: NewVolumeFormat("raw", false)},
			`name: foobar_baz
driver: foo

name: foobar_bar
driver: bar

`,
		},
		{
			Context{Format: NewVolumeFormat("raw", true)},
			`name: foobar_baz
name: foobar_bar
`,
		},
		// Custom Format
		{
			Context{Format: NewVolumeFormat("{{.Name}}", false)},
			`foobar_baz
foobar_bar
`,
		},
	}

	volumes := []*volume.Volume{
		{Name: "foobar_baz", Driver: "foo"},
		{Name: "foobar_bar", Driver: "bar"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(string(tc.context.Format), func(t *testing.T) {
			var out bytes.Buffer
			tc.context.Output = &out
			err := VolumeWrite(tc.context, volumes)
			if err != nil {
				assert.Error(t, err, tc.expected)
			} else {
				assert.Equal(t, out.String(), tc.expected)
			}
		})
	}
}

func TestVolumeContextWriteJSON(t *testing.T) {
	volumes := []*volume.Volume{
		{Driver: "foo", Name: "foobar_baz"},
		{Driver: "bar", Name: "foobar_bar"},
	}
	expectedJSONs := []map[string]interface{}{
		{"Availability": "N/A", "Driver": "foo", "Group": "N/A", "Labels": "", "Links": "N/A", "Mountpoint": "", "Name": "foobar_baz", "Scope": "", "Size": "N/A", "Status": "N/A"},
		{"Availability": "N/A", "Driver": "bar", "Group": "N/A", "Labels": "", "Links": "N/A", "Mountpoint": "", "Name": "foobar_bar", "Scope": "", "Size": "N/A", "Status": "N/A"},
	}
	out := bytes.NewBufferString("")
	err := VolumeWrite(Context{Format: "{{json .}}", Output: out}, volumes)
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

func TestVolumeContextWriteJSONField(t *testing.T) {
	volumes := []*volume.Volume{
		{Driver: "foo", Name: "foobar_baz"},
		{Driver: "bar", Name: "foobar_bar"},
	}
	out := bytes.NewBufferString("")
	err := VolumeWrite(Context{Format: "{{json .Name}}", Output: out}, volumes)
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		msg := fmt.Sprintf("Output: line %d: %s", i, line)
		var s string
		err := json.Unmarshal([]byte(line), &s)
		assert.NilError(t, err, msg)
		assert.Check(t, is.Equal(volumes[i].Name, s), msg)
	}
}
