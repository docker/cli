package container

import (
	"bytes"
	"testing"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/archive"
	"gotest.tools/v3/assert"
)

func TestDiffContextFormatWrite(t *testing.T) {
	// Check default output format (verbose and non-verbose mode) for table headers
	cases := []struct {
		context  formatter.Context
		expected string
	}{
		{
			formatter.Context{Format: NewDiffFormat("table")},
			`CHANGE TYPE   PATH
C             /var/log/app.log
A             /usr/app/app.js
D             /usr/app/old_app.js
`,
		},
		{
			formatter.Context{Format: NewDiffFormat("table {{.Path}}")},
			`PATH
/var/log/app.log
/usr/app/app.js
/usr/app/old_app.js
`,
		},
		{
			formatter.Context{Format: NewDiffFormat("{{.Type}}: {{.Path}}")},
			`C: /var/log/app.log
A: /usr/app/app.js
D: /usr/app/old_app.js
`,
		},
	}

	diffs := []container.ContainerChangeResponseItem{
		{Kind: archive.ChangeModify, Path: "/var/log/app.log"},
		{Kind: archive.ChangeAdd, Path: "/usr/app/app.js"},
		{Kind: archive.ChangeDelete, Path: "/usr/app/old_app.js"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(string(tc.context.Format), func(t *testing.T) {
			out := bytes.NewBufferString("")
			tc.context.Output = out
			err := DiffFormatWrite(tc.context, diffs)
			if err != nil {
				assert.Error(t, err, tc.expected)
			} else {
				assert.Equal(t, out.String(), tc.expected)
			}
		})
	}
}
