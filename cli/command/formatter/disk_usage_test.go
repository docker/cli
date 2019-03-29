package formatter

import (
	"bytes"
	"testing"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"gotest.tools/golden"
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
					Format: NewDiskUsageFormat("table", false),
				},
				Verbose: false},
			`TYPE                TOTAL               ACTIVE              SIZE                RECLAIMABLE
Images              0                   0                   0B                  0B
Containers          0                   0                   0B                  0B
Local Volumes       0                   0                   0B                  0B
Build Cache         0                   0                   0B                  0B
`,
		},
		{
			DiskUsageContext{Verbose: true, Context: Context{Format: NewDiskUsageFormat("table", true)}},
			`Images space usage:

REPOSITORY          TAG                 IMAGE ID            CREATED             SIZE                SHARED SIZE         UNIQUE SIZE         CONTAINERS

Containers space usage:

CONTAINER ID        IMAGE               COMMAND             LOCAL VOLUMES       SIZE                CREATED             STATUS              NAMES

Local Volumes space usage:

VOLUME NAME         LINKS               SIZE

Build cache usage: 0B

CACHE ID            CACHE TYPE          SIZE                CREATED             LAST USED           USAGE               SHARED
`,
		},
		{
			DiskUsageContext{Verbose: true, Context: Context{Format: NewDiskUsageFormat("raw", true)}},
			``,
		},
		{
			DiskUsageContext{Verbose: true, Context: Context{Format: NewDiskUsageFormat("{{json .}}", true)}},
			`{"Images":[],"Containers":[],"Volumes":[],"BuildCache":[]}`,
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
					Format: NewDiskUsageFormat("table", false),
				},
			},
			`TYPE                TOTAL               ACTIVE              SIZE                RECLAIMABLE
Images              0                   0                   0B                  0B
Containers          0                   0                   0B                  0B
Local Volumes       0                   0                   0B                  0B
Build Cache         0                   0                   0B                  0B
`,
		},
		{
			DiskUsageContext{
				Context: Context{
					Format: NewDiskUsageFormat("table {{.Type}}\t{{.Active}}", false),
				},
			},
			string(golden.Get(t, "disk-usage-context-write-custom.golden")),
		},
		// Raw Format
		{
			DiskUsageContext{
				Context: Context{
					Format: NewDiskUsageFormat("raw", false),
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
