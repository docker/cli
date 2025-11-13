package image

import (
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"gotest.tools/v3/assert"
)

func TestPrintImageTreeAnsiTty(t *testing.T) {
	testCases := []struct {
		name         string
		stdinTty     bool
		stdoutTty    bool
		stderrTty    bool
		expectedAnsi bool
		noColorEnv   bool
	}{
		{
			name:      "non-terminal",
			stdinTty:  false,
			stdoutTty: false,
			stderrTty: false,

			expectedAnsi: false,
		},
		{
			name:      "terminal",
			stdinTty:  true,
			stdoutTty: true,
			stderrTty: true,

			expectedAnsi: true,
		},
		{
			name:      "stdout-tty-only",
			stdinTty:  false,
			stdoutTty: true,
			stderrTty: false,

			expectedAnsi: true,
		},
		{
			name:      "stdin-stderr-tty-only",
			stdinTty:  true,
			stdoutTty: false,
			stderrTty: true,

			expectedAnsi: false,
		},
		{
			name:      "stdout-stdin-tty",
			stdinTty:  true,
			stdoutTty: true,
			stderrTty: false,

			expectedAnsi: true,
		},
		{
			name:      "stdout-stderr-tty",
			stdinTty:  false,
			stdoutTty: true,
			stderrTty: true,

			expectedAnsi: true,
		},
		{
			name:      "stdin-tty-only",
			stdinTty:  true,
			stdoutTty: false,
			stderrTty: false,

			expectedAnsi: false,
		},
		{
			name:      "stderr-tty-only",
			stdinTty:  false,
			stdoutTty: false,
			stderrTty: true,

			expectedAnsi: false,
		},
		{
			name:      "no-color-env",
			stdinTty:  false,
			stdoutTty: false,
			stderrTty: false,

			noColorEnv:   true,
			expectedAnsi: false,
		},
		{
			name:      "no-color-env-terminal",
			stdinTty:  true,
			stdoutTty: true,
			stderrTty: true,

			noColorEnv:   true,
			expectedAnsi: false,
		},
	}

	mockView := treeView{
		images: []topImage{
			{
				Names: []string{"test-image:latest"},
				Details: imageDetails{
					ID:          "sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					DiskUsage:   "10.5 MB",
					InUse:       true,
					ContentSize: "5.2 MB",
				},
				Children: []subImage{
					{
						Platform:  "linux/amd64",
						Available: true,
						Details: imageDetails{
							ID:          "sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
							DiskUsage:   "5.1 MB",
							InUse:       false,
							ContentSize: "2.5 MB",
						},
					},
				},
			},
		},
		imageSpacing: false,
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(nil)
			cli.In().SetIsTerminal(tc.stdinTty)
			cli.Out().SetIsTerminal(tc.stdoutTty)
			cli.Err().SetIsTerminal(tc.stderrTty)
			if tc.noColorEnv {
				t.Setenv("NO_COLOR", "1")
			} else {
				t.Setenv("NO_COLOR", "")
			}

			printImageTree(cli, mockView)

			out := cli.OutBuffer().String()
			assert.Check(t, len(out) > 0, "Output should not be empty")

			hasAnsi := strings.Contains(out, "\x1b[")
			if tc.expectedAnsi {
				assert.Check(t, hasAnsi, "Output should contain ANSI escape codes, output: %s", out)
			} else {
				assert.Check(t, !hasAnsi, "Output should not contain ANSI escape codes, output: %s", out)
			}
		})
	}
}
