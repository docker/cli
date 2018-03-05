package formatter

import (
	"bytes"
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/gotestyourself/gotestyourself/golden"
)

func TestDiskUsageContextFormatWrite(t *testing.T) {
	cases := []struct {
		context  DiskUsageContext
		expected string
	}{
		// Check default output format (verbose and non-verbose mode) for table headers
		{
			DiskUsageContext{
				Context: Context{
					Format: NewDiskUsageFormat("table"),
				},
				Verbose: false},
			`TYPE                TOTAL               ACTIVE              SIZE                RECLAIMABLE
Images              0                   0                   0B                  0B
Containers          0                   0                   0B                  0B
Local Volumes       0                   0                   0B                  0B
Build Cache                                                 0B                  0B
`,
		},
		{
			DiskUsageContext{Verbose: true},
			`Images space usage:

REPOSITORY          TAG                 IMAGE ID            CREATED ago         SIZE                SHARED SIZE         UNIQUE SiZE         CONTAINERS

Containers space usage:

CONTAINER ID        IMAGE               COMMAND             LOCAL VOLUMES       SIZE                CREATED ago         STATUS              NAMES

Local Volumes space usage:

VOLUME NAME         LINKS               SIZE

Build cache usage: 0B

`,
		},
		// Errors
		{
			DiskUsageContext{
				Context: Context{
					Format: "{{InvalidFunction}}",
				},
			},
			`Template parsing error: template: :1: function "InvalidFunction" not defined
`,
		},
		{
			DiskUsageContext{
				Context: Context{
					Format: "{{nil}}",
				},
			},
			`Template parsing error: template: :1:2: executing "" at <nil>: nil is not a command
`,
		},
		// Table Format
		{
			DiskUsageContext{
				Context: Context{
					Format: NewDiskUsageFormat("table"),
				},
			},
			`TYPE                TOTAL               ACTIVE              SIZE                RECLAIMABLE
Images              0                   0                   0B                  0B
Containers          0                   0                   0B                  0B
Local Volumes       0                   0                   0B                  0B
Build Cache                                                 0B                  0B
`,
		},
		{
			DiskUsageContext{
				Context: Context{
					Format: NewDiskUsageFormat("table {{.Type}}\t{{.Active}}"),
				},
			},
			string(golden.Get(t, "disk-usage-context-write-custom.golden")),
		},
		// Raw Format
		{
			DiskUsageContext{
				Context: Context{
					Format: NewDiskUsageFormat("raw"),
				},
			},
			string(golden.Get(t, "disk-usage-raw-format.golden")),
		},
	}

	for _, testcase := range cases {
		out := bytes.NewBufferString("")
		testcase.context.Output = out
		if err := testcase.context.Write(); err != nil {
			assert.Check(t, is.Equal(testcase.expected, err.Error()))
		} else {
			assert.Check(t, is.Equal(testcase.expected, out.String()))
		}
	}
}
