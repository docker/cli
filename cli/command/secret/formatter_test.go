package secret

import (
	"bytes"
	"testing"
	"time"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/docker/api/types/swarm"
	"gotest.tools/v3/assert"
)

func TestSecretContextFormatWrite(t *testing.T) {
	// Check default output format (verbose and non-verbose mode) for table headers
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
		{formatter.Context{Format: NewFormat("table", false)},
			`ID        NAME        DRIVER    CREATED                  UPDATED
1         passwords             Less than a second ago   Less than a second ago
2         id_rsa                Less than a second ago   Less than a second ago
`},
		{formatter.Context{Format: NewFormat("table {{.Name}}", true)},
			`NAME
passwords
id_rsa
`},
		{formatter.Context{Format: NewFormat("{{.ID}}-{{.Name}}", false)},
			`1-passwords
2-id_rsa
`},
	}

	secrets := []swarm.Secret{
		{ID: "1",
			Meta: swarm.Meta{CreatedAt: time.Now(), UpdatedAt: time.Now()},
			Spec: swarm.SecretSpec{Annotations: swarm.Annotations{Name: "passwords"}}},
		{ID: "2",
			Meta: swarm.Meta{CreatedAt: time.Now(), UpdatedAt: time.Now()},
			Spec: swarm.SecretSpec{Annotations: swarm.Annotations{Name: "id_rsa"}}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(string(tc.context.Format), func(t *testing.T) {
			var out bytes.Buffer
			tc.context.Output = &out

			if err := FormatWrite(tc.context, secrets); err != nil {
				assert.Error(t, err, tc.expected)
			} else {
				assert.Equal(t, out.String(), tc.expected)
			}
		})
	}
}
