package image

import (
	"fmt"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
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

func TestPrintImageTreeGolden(t *testing.T) {
	testCases := []struct {
		name     string
		view     treeView
		expanded bool
	}{
		{
			name:     "width-calculation-untagged",
			expanded: false,
			view: treeView{
				images: []topImage{
					{
						Names: []string{"a:1"},
						Details: imageDetails{
							ID:          "sha256:1111111111111111111111111111111111111111111111111111111111111111",
							DiskUsage:   "5.5 MB",
							InUse:       false,
							ContentSize: "2.5 MB",
						},
					},
					{
						// Untagged image name is longer than "a:1"
						Names: []string{},
						Details: imageDetails{
							ID:          "sha256:2222222222222222222222222222222222222222222222222222222222222222",
							DiskUsage:   "3.2 MB",
							InUse:       false,
							ContentSize: "1.6 MB",
						},
					},
					{
						Names: []string{"short:v1"},
						Details: imageDetails{
							ID:          "sha256:3333333333333333333333333333333333333333333333333333333333333333",
							DiskUsage:   "7.1 MB",
							InUse:       true,
							ContentSize: "3.5 MB",
						},
					},
				},
				imageSpacing: false,
			},
		},
		{
			name:     "expanded-view-with-platforms",
			expanded: false,
			view: treeView{
				images: []topImage{
					{
						Names: []string{"multiplatform:latest"},
						Details: imageDetails{
							ID:          "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
							DiskUsage:   "25.5 MB",
							InUse:       true,
							ContentSize: "20.2 MB",
						},
						Children: []subImage{
							{
								Platform:  "linux/amd64",
								Available: true,
								Details: imageDetails{
									ID:          "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
									DiskUsage:   "12.1 MB",
									InUse:       false,
									ContentSize: "10.0 MB",
								},
							},
							{
								Platform:  "linux/arm64",
								Available: true,
								Details: imageDetails{
									ID:          "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
									DiskUsage:   "13.4 MB",
									InUse:       true,
									ContentSize: "10.2 MB",
								},
							},
						},
					},
				},
				imageSpacing: true,
			},
		},
		{
			name:     "untagged-with-platforms",
			expanded: false,
			view: treeView{
				images: []topImage{
					{
						Names: []string{},
						Details: imageDetails{
							ID:          "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
							DiskUsage:   "18.5 MB",
							InUse:       false,
							ContentSize: "15.2 MB",
						},
						Children: []subImage{
							{
								Platform:  "linux/amd64",
								Available: true,
								Details: imageDetails{
									ID:          "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
									DiskUsage:   "9.2 MB",
									InUse:       false,
									ContentSize: "7.6 MB",
								},
							},
							{
								Platform:  "linux/arm64",
								Available: false,
								Details: imageDetails{
									ID:          "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
									DiskUsage:   "9.3 MB",
									InUse:       false,
									ContentSize: "7.6 MB",
								},
							},
						},
					},
				},
				imageSpacing: true,
			},
		},
		{
			name:     "mixed-tagged-untagged-with-children",
			expanded: false,
			view: treeView{
				images: []topImage{
					{
						Names: []string{"app:v1", "app:latest"},
						Details: imageDetails{
							ID:          "sha256:1010101010101010101010101010101010101010101010101010101010101010",
							DiskUsage:   "30.5 MB",
							InUse:       true,
							ContentSize: "25.2 MB",
						},
						Children: []subImage{
							{
								Platform:  "linux/amd64",
								Available: true,
								Details: imageDetails{
									ID:          "sha256:2020202020202020202020202020202020202020202020202020202020202020",
									DiskUsage:   "15.2 MB",
									InUse:       true,
									ContentSize: "12.6 MB",
								},
							},
						},
					},
					{
						Names: []string{},
						Details: imageDetails{
							ID:          "sha256:3030303030303030303030303030303030303030303030303030303030303030",
							DiskUsage:   "12.3 MB",
							InUse:       false,
							ContentSize: "10.1 MB",
						},
						Children: []subImage{
							{
								Platform:  "linux/arm/v7",
								Available: true,
								Details: imageDetails{
									ID:          "sha256:4040404040404040404040404040404040404040404040404040404040404040",
									DiskUsage:   "6.1 MB",
									InUse:       false,
									ContentSize: "5.0 MB",
								},
							},
						},
					},
					{
						Names: []string{"base:alpine"},
						Details: imageDetails{
							ID:          "sha256:5050505050505050505050505050505050505050505050505050505050505050",
							DiskUsage:   "5.5 MB",
							InUse:       false,
							ContentSize: "5.5 MB",
						},
					},
				},
				imageSpacing: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(nil)
			cli.Out().SetIsTerminal(false)

			printImageTree(cli, tc.view)

			golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("tree-command-success.%s.golden", tc.name))
		})
	}
}
