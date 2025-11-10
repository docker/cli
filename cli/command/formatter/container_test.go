// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.24

package formatter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/api/types/container"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/golden"
)

func TestContainerPsContext(t *testing.T) {
	containerID := test.RandomID()
	unix := time.Now().Add(-65 * time.Second).Unix()

	var ctx ContainerContext
	cases := []struct {
		container container.Summary
		trunc     bool
		expValue  string
		call      func() string
	}{
		{
			container: container.Summary{ID: containerID},
			trunc:     true,
			expValue:  TruncateID(containerID),
			call:      ctx.ID,
		},
		{
			container: container.Summary{ID: containerID},
			expValue:  containerID,
			call:      ctx.ID,
		},
		{
			container: container.Summary{Names: []string{"/foobar_baz"}},
			trunc:     true,
			expValue:  "foobar_baz",
			call:      ctx.Names,
		},
		{
			container: container.Summary{Image: "ubuntu"},
			trunc:     true,
			expValue:  "ubuntu",
			call:      ctx.Image,
		},
		{
			container: container.Summary{Image: "ubuntu:latest"},
			trunc:     true,
			expValue:  "ubuntu:latest",
			call:      ctx.Image,
		},
		{
			container: container.Summary{Image: "docker.io/library/ubuntu"},
			trunc:     true,
			expValue:  "ubuntu",
			call:      ctx.Image,
		},
		{
			container: container.Summary{Image: "docker.io/library/ubuntu:latest"},
			trunc:     true,
			expValue:  "ubuntu:latest",
			call:      ctx.Image,
		},
		{
			container: container.Summary{Image: "ubuntu:latest@sha256:a5a665ff33eced1e0803148700880edab4269067ed77e27737a708d0d293fbf5"},
			trunc:     true,
			expValue:  "ubuntu:latest",
			call:      ctx.Image,
		},
		{
			container: container.Summary{Image: "ubuntu@sha256:a5a665ff33eced1e0803148700880edab4269067ed77e27737a708d0d293fbf5"},
			trunc:     true,
			expValue:  "ubuntu",
			call:      ctx.Image,
		},
		{
			container: container.Summary{Image: "docker.io/library/ubuntu@sha256:a5a665ff33eced1e0803148700880edab4269067ed77e27737a708d0d293fbf5"},
			trunc:     true,
			expValue:  "ubuntu",
			call:      ctx.Image,
		},
		{
			container: container.Summary{Image: "docker.io/library/ubuntu:latest@sha256:a5a665ff33eced1e0803148700880edab4269067ed77e27737a708d0d293fbf5"},
			trunc:     true,
			expValue:  "ubuntu:latest",
			call:      ctx.Image,
		},
		{
			container: container.Summary{Image: "verylongimagenameverylongimagenameverylongimagenameverylongimagenameverylongimagenameverylongimagenameverylongimagename"},
			trunc:     true,
			expValue:  "verylongimagenameverylongimagenameverylongimagenameverylongimagenameverylongimagenameverylongimagenameverylongimagename",
			call:      ctx.Image,
		},
		{
			container: container.Summary{
				Image:   "a5a665ff33eced1e0803148700880edab4",
				ImageID: "a5a665ff33eced1e0803148700880edab4269067ed77e27737a708d0d293fbf5",
			},
			trunc:    true,
			expValue: "a5a665ff33ec",
			call:     ctx.Image,
		},
		{
			container: container.Summary{
				Image:   "a5a665ff33eced1e0803148700880edab4",
				ImageID: "a5a665ff33eced1e0803148700880edab4269067ed77e27737a708d0d293fbf5",
			},
			expValue: "a5a665ff33eced1e0803148700880edab4",
			call:     ctx.Image,
		},
		{
			container: container.Summary{Image: ""},
			trunc:     true,
			expValue:  "<no image>",
			call:      ctx.Image,
		},
		{
			container: container.Summary{Command: "sh -c 'ls -la'"},
			trunc:     true,
			expValue:  `"sh -c 'ls -la'"`,
			call:      ctx.Command,
		},
		{
			container: container.Summary{Created: unix},
			trunc:     true,
			expValue:  time.Unix(unix, 0).String(),
			call:      ctx.CreatedAt,
		},
		{
			container: container.Summary{Ports: []container.PortSummary{{PrivatePort: 8080, PublicPort: 8080, Type: "tcp"}}},
			trunc:     true,
			expValue:  "8080/tcp",
			call:      ctx.Ports,
		},
		{
			container: container.Summary{Status: "Up 123 seconds"},
			trunc:     true,
			expValue:  "Up 123 seconds",
			call:      ctx.Status,
		},
		{
			container: container.Summary{State: container.StateRunning},
			trunc:     true,
			expValue:  string(container.StateRunning),
			call:      ctx.State,
		},
		{
			container: container.Summary{SizeRw: 10},
			trunc:     true,
			expValue:  "10B",
			call:      ctx.Size,
		},
		{
			container: container.Summary{SizeRw: 10, SizeRootFs: 20},
			trunc:     true,
			expValue:  "10B (virtual 20B)",
			call:      ctx.Size,
		},
		{
			container: container.Summary{},
			trunc:     true,
			call:      ctx.Labels,
		},
		{
			container: container.Summary{Labels: map[string]string{"cpu": "6", "storage": "ssd"}},
			trunc:     true,
			expValue:  "cpu=6,storage=ssd",
			call:      ctx.Labels,
		},
		{
			container: container.Summary{Created: unix},
			trunc:     true,
			expValue:  "About a minute ago",
			call:      ctx.RunningFor,
		},
		{
			container: container.Summary{
				Mounts: []container.MountPoint{
					{
						Name:   "this-is-a-long-volume-name-and-will-be-truncated-if-trunc-is-set",
						Driver: "local",
						Source: "/a/path",
					},
				},
			},
			trunc:    true,
			expValue: "this-is-a-long…",
			call:     ctx.Mounts,
		},
		{
			container: container.Summary{
				Mounts: []container.MountPoint{
					{
						Driver: "local",
						Source: "/a/path",
					},
				},
			},
			expValue: "/a/path",
			call:     ctx.Mounts,
		},
		{
			container: container.Summary{
				Mounts: []container.MountPoint{
					{
						Name:   "733908409c91817de8e92b0096373245f329f19a88e2c849f02460e9b3d1c203",
						Driver: "local",
						Source: "/a/path",
					},
				},
			},
			expValue: "733908409c91817de8e92b0096373245f329f19a88e2c849f02460e9b3d1c203",
			call:     ctx.Mounts,
		},
	}

	for _, c := range cases {
		ctx = ContainerContext{c: c.container, trunc: c.trunc}
		v := c.call()
		if strings.Contains(v, ",") {
			test.CompareMultipleValues(t, v, c.expValue)
		} else if v != c.expValue {
			t.Fatalf("Expected %s, was %s\n", c.expValue, v)
		}
	}

	c1 := container.Summary{Labels: map[string]string{"com.docker.swarm.swarm-id": "33", "com.docker.swarm.node_name": "ubuntu"}}
	ctx = ContainerContext{c: c1, trunc: true}

	sid := ctx.Label("com.docker.swarm.swarm-id")
	node := ctx.Label("com.docker.swarm.node_name")
	if sid != "33" {
		t.Fatalf("Expected 33, was %s\n", sid)
	}

	if node != "ubuntu" {
		t.Fatalf("Expected ubuntu, was %s\n", node)
	}

	c2 := container.Summary{}
	ctx = ContainerContext{c: c2, trunc: true}

	label := ctx.Label("anything.really")
	if label != "" {
		t.Fatalf("Expected an empty string, was %s", label)
	}
}

func TestContainerContextWrite(t *testing.T) {
	unixTime := time.Now().AddDate(0, 0, -1).Unix()
	expectedTime := time.Unix(unixTime, 0).String()

	cases := []struct {
		context  Context
		expected string
	}{
		// Errors
		{
			context:  Context{Format: "{{InvalidFunction}}"},
			expected: `template parsing error: template: :1: function "InvalidFunction" not defined`,
		},
		{
			context:  Context{Format: "{{nil}}"},
			expected: `template parsing error: template: :1:2: executing "" at <nil>: nil is not a command`,
		},
		// Table Format
		{
			context: Context{Format: NewContainerFormat("table", false, true)},
			expected: `CONTAINER ID   IMAGE     COMMAND   CREATED        STATUS    PORTS     NAMES        SIZE
containerID1   ubuntu    ""        24 hours ago                       foobar_baz   0B
containerID2   ubuntu    ""        24 hours ago                       foobar_bar   0B
`,
		},
		{
			context: Context{Format: NewContainerFormat("table", false, false)},
			expected: `CONTAINER ID   IMAGE     COMMAND   CREATED        STATUS    PORTS     NAMES
containerID1   ubuntu    ""        24 hours ago                       foobar_baz
containerID2   ubuntu    ""        24 hours ago                       foobar_bar
`,
		},
		{
			context:  Context{Format: NewContainerFormat("table {{.Image}}", false, false)},
			expected: "IMAGE\nubuntu\nubuntu\n",
		},
		{
			context:  Context{Format: NewContainerFormat("table {{.Image}}", false, true)},
			expected: "IMAGE\nubuntu\nubuntu\n",
		},
		{
			context:  Context{Format: NewContainerFormat("table {{.Image}}", true, false)},
			expected: "containerID1\ncontainerID2\n",
		},
		{
			context:  Context{Format: NewContainerFormat("table", true, false)},
			expected: "containerID1\ncontainerID2\n",
		},
		{
			context:  Context{Format: NewContainerFormat("table {{.State}}", false, true)},
			expected: "STATE\nrunning\nrunning\n",
		},
		// Raw Format
		{
			context: Context{Format: NewContainerFormat("raw", false, false)},
			expected: fmt.Sprintf(`container_id: containerID1
image: ubuntu
command: ""
created_at: %s
state: running
status:
names: foobar_baz
labels:
ports:

container_id: containerID2
image: ubuntu
command: ""
created_at: %s
state: running
status:
names: foobar_bar
labels:
ports:

`, expectedTime, expectedTime),
		},
		{
			context: Context{Format: NewContainerFormat("raw", false, true)},
			expected: fmt.Sprintf(`container_id: containerID1
image: ubuntu
command: ""
created_at: %s
state: running
status:
names: foobar_baz
labels:
ports:
size: 0B

container_id: containerID2
image: ubuntu
command: ""
created_at: %s
state: running
status:
names: foobar_bar
labels:
ports:
size: 0B

`, expectedTime, expectedTime),
		},
		{
			context:  Context{Format: NewContainerFormat("raw", true, false)},
			expected: "container_id: containerID1\ncontainer_id: containerID2\n",
		},
		// Custom Format
		{
			context:  Context{Format: "{{.Image}}"},
			expected: "ubuntu\nubuntu\n",
		},
		{
			context:  Context{Format: NewContainerFormat("{{.Image}}", false, true)},
			expected: "ubuntu\nubuntu\n",
		},
		// Special headers for customized table format
		{
			context: Context{
				Format: NewContainerFormat(
					`table {{truncate .ID 5}}\t{{json .Image}} {{.RunningFor}}/{{title .Status}}/{{pad .Ports 2 2}}.{{upper .Names}} {{lower .Status}}`,
					false, true,
				),
			},
			expected: string(golden.Get(t, "container-context-write-special-headers.golden")),
		},
		{
			context:  Context{Format: NewContainerFormat(`table {{split .Image ":"}}`, false, false)},
			expected: "IMAGE\n[ubuntu]\n[ubuntu]\n",
		},
	}

	containers := []container.Summary{
		{ID: "containerID1", Names: []string{"/foobar_baz"}, Image: "ubuntu", Created: unixTime, State: container.StateRunning},
		{ID: "containerID2", Names: []string{"/foobar_bar"}, Image: "ubuntu", Created: unixTime, State: container.StateRunning},
	}

	for _, tc := range cases {
		t.Run(string(tc.context.Format), func(t *testing.T) {
			var out bytes.Buffer
			tc.context.Output = &out
			err := ContainerWrite(tc.context, containers)
			if err != nil {
				assert.Error(t, err, tc.expected)
			} else {
				assert.Equal(t, out.String(), tc.expected)
			}
		})
	}
}

func TestContainerContextWriteWithNoContainers(t *testing.T) {
	cases := []struct {
		context  Context
		expected string
	}{
		{
			context: Context{
				Format: "{{.Image}}",
			},
		},
		{
			context: Context{
				Format: "table {{.Image}}",
			},
			expected: "IMAGE\n",
		},
		{
			context: Context{
				Format: NewContainerFormat("{{.Image}}", false, true),
			},
		},
		{
			context: Context{
				Format: NewContainerFormat("table {{.Image}}", false, true),
			},
			expected: "IMAGE\n",
		},
		{
			context: Context{
				Format: "table {{.Image}}\t{{.Size}}",
			},
			expected: "IMAGE     SIZE\n",
		},
		{
			context: Context{
				Format: NewContainerFormat("table {{.Image}}\t{{.Size}}", false, true),
			},
			expected: "IMAGE     SIZE\n",
		},
	}

	for _, tc := range cases {
		t.Run(string(tc.context.Format), func(t *testing.T) {
			out := new(bytes.Buffer)
			tc.context.Output = out
			err := ContainerWrite(tc.context, nil)
			assert.NilError(t, err)
			assert.Equal(t, out.String(), tc.expected)
		})
	}
}

func TestContainerContextWriteJSON(t *testing.T) {
	unix := time.Now().Add(-65 * time.Second).Unix()
	containers := []container.Summary{
		{
			ID:      "containerID1",
			Names:   []string{"/foobar_baz"},
			Image:   "ubuntu",
			Created: unix,
			State:   container.StateRunning,
		},
		{
			ID:      "containerID2",
			Names:   []string{"/foobar_bar"},
			Image:   "ubuntu",
			Created: unix,
			State:   container.StateRunning,

			ImageManifestDescriptor: &ocispec.Descriptor{Platform: &ocispec.Platform{Architecture: "amd64", OS: "linux"}},
		},
		{
			ID:      "containerID3",
			Names:   []string{"/foobar_bar"},
			Image:   "ubuntu",
			Created: unix,
			State:   container.StateRunning,

			ImageManifestDescriptor: &ocispec.Descriptor{Platform: &ocispec.Platform{}},
		},
	}
	expectedCreated := time.Unix(unix, 0).String()
	expectedJSONs := []map[string]any{
		{
			"Command":      `""`,
			"CreatedAt":    expectedCreated,
			"ID":           "containerID1",
			"Image":        "ubuntu",
			"Labels":       "",
			"LocalVolumes": "0",
			"Mounts":       "",
			"Names":        "foobar_baz",
			"Networks":     "",
			"Platform":     nil,
			"Ports":        "",
			"RunningFor":   "About a minute ago",
			"Size":         "0B",
			"State":        "running",
			"Status":       "",
		},
		{
			"Command":      `""`,
			"CreatedAt":    expectedCreated,
			"ID":           "containerID2",
			"Image":        "ubuntu",
			"Labels":       "",
			"LocalVolumes": "0",
			"Mounts":       "",
			"Names":        "foobar_bar",
			"Networks":     "",
			"Platform":     map[string]any{"architecture": "amd64", "os": "linux"},
			"Ports":        "",
			"RunningFor":   "About a minute ago",
			"Size":         "0B",
			"State":        "running",
			"Status":       "",
		},
		{
			"Command":      `""`,
			"CreatedAt":    expectedCreated,
			"ID":           "containerID3",
			"Image":        "ubuntu",
			"Labels":       "",
			"LocalVolumes": "0",
			"Mounts":       "",
			"Names":        "foobar_bar",
			"Networks":     "",
			"Platform":     map[string]any{"architecture": "", "os": ""},
			"Ports":        "",
			"RunningFor":   "About a minute ago",
			"Size":         "0B",
			"State":        "running",
			"Status":       "",
		},
	}
	out := bytes.NewBufferString("")
	err := ContainerWrite(Context{Format: "{{json .}}", Output: out}, containers)
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		msg := fmt.Sprintf("Output: line %d: %s", i, line)
		var m map[string]any
		err := json.Unmarshal([]byte(line), &m)
		assert.NilError(t, err, msg)
		assert.Check(t, is.DeepEqual(expectedJSONs[i], m), msg)
	}
}

func TestContainerContextWriteJSONField(t *testing.T) {
	containers := []container.Summary{
		{ID: "containerID1", Names: []string{"/foobar_baz"}, Image: "ubuntu"},
		{ID: "containerID2", Names: []string{"/foobar_bar"}, Image: "ubuntu"},
	}
	out := bytes.NewBufferString("")
	err := ContainerWrite(Context{Format: "{{json .ID}}", Output: out}, containers)
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		msg := fmt.Sprintf("Output: line %d: %s", i, line)
		var s string
		err := json.Unmarshal([]byte(line), &s)
		assert.NilError(t, err, msg)
		assert.Check(t, is.Equal(containers[i].ID, s), msg)
	}
}

func TestContainerBackCompat(t *testing.T) {
	createdAtTime := time.Now().AddDate(-1, 0, 0) // 1 year ago

	ctrContext := container.Summary{
		ID:                      "aabbccddeeff",
		Names:                   []string{"/foobar_baz"},
		Image:                   "docker.io/library/ubuntu",                                                // should this have canonical format or not?
		ImageID:                 "sha256:a5a665ff33eced1e0803148700880edab4269067ed77e27737a708d0d293fbf5", // should this have algo-prefix or not?
		ImageManifestDescriptor: nil,
		Command:                 "/bin/sh",
		Created:                 createdAtTime.UTC().Unix(),
		Ports:                   []container.PortSummary{{PrivatePort: 8080, PublicPort: 8080, Type: "tcp"}},
		SizeRw:                  123,
		SizeRootFs:              12345,
		Labels:                  map[string]string{"label1": "value1", "label2": "value2"},
		State:                   "running",
		Status:                  "running",
		HostConfig: struct {
			NetworkMode string            `json:",omitempty"`
			Annotations map[string]string `json:",omitempty"`
		}{
			NetworkMode: "bridge",
			Annotations: map[string]string{
				"com.example.annotation": "hello",
			},
		},
		NetworkSettings: nil,
		Mounts:          nil,
	}

	tests := []struct {
		field    string
		expected string
	}{
		{field: "ID", expected: "aabbccddeeff"},
		{field: "Names", expected: "foobar_baz"},
		{field: "Image", expected: "docker.io/library/ubuntu"},
		{field: "Command", expected: `"/bin/sh"`},
		{field: "CreatedAt", expected: time.Unix(createdAtTime.Unix(), 0).String()},
		{field: "RunningFor", expected: "12 months ago"},
		{field: "Ports", expected: "8080/tcp"},
		{field: "Status", expected: "running"},
		{field: "Size", expected: "123B (virtual 12.3kB)"},
		{field: "Labels", expected: "label1=value1,label2=value2"},
		{field: "Mounts", expected: ""},
	}

	for _, tc := range tests {
		t.Run(tc.field, func(t *testing.T) {
			buf := new(bytes.Buffer)
			ctx := Context{Format: Format(fmt.Sprintf("{{ .%s }}", tc.field)), Output: buf}
			assert.NilError(t, ContainerWrite(ctx, []container.Summary{ctrContext}))
			assert.Check(t, is.Equal(strings.TrimSpace(buf.String()), tc.expected))
		})
	}
}

type ports struct {
	ports    []container.PortSummary
	expected string
}

func TestDisplayablePorts(t *testing.T) {
	cases := []ports{
		{
			ports: []container.PortSummary{
				{
					PrivatePort: 9988,
					Type:        "tcp",
				},
			},
			expected: "9988/tcp",
		},
		{
			ports: []container.PortSummary{
				{
					PrivatePort: 9988,
					Type:        "udp",
				},
			},
			expected: "9988/udp",
		},
		{
			ports: []container.PortSummary{
				{
					IP:          netip.MustParseAddr("0.0.0.0"),
					PrivatePort: 9988,
					Type:        "tcp",
				},
			},
			expected: "0.0.0.0:0->9988/tcp",
		},
		{
			ports: []container.PortSummary{
				{
					IP:          netip.MustParseAddr("::"),
					PrivatePort: 9988,
					Type:        "tcp",
				},
			},
			expected: "[::]:0->9988/tcp",
		},
		{
			ports: []container.PortSummary{
				{
					PrivatePort: 9988,
					PublicPort:  8899,
					Type:        "tcp",
				},
			},
			expected: "9988/tcp",
		},
		{
			ports: []container.PortSummary{
				{
					IP:          netip.MustParseAddr("4.3.2.1"),
					PrivatePort: 9988,
					PublicPort:  8899,
					Type:        "tcp",
				},
			},
			expected: "4.3.2.1:8899->9988/tcp",
		},
		{
			ports: []container.PortSummary{
				{
					IP:          netip.MustParseAddr("::1"),
					PrivatePort: 9988,
					PublicPort:  8899,
					Type:        "tcp",
				},
			},
			expected: "[::1]:8899->9988/tcp",
		},
		{
			ports: []container.PortSummary{
				{
					IP:          netip.MustParseAddr("4.3.2.1"),
					PrivatePort: 9988,
					PublicPort:  9988,
					Type:        "tcp",
				},
			},
			expected: "4.3.2.1:9988->9988/tcp",
		},
		{
			ports: []container.PortSummary{
				{
					IP:          netip.MustParseAddr("::1"),
					PrivatePort: 9988,
					PublicPort:  9988,
					Type:        "tcp",
				},
			},
			expected: "[::1]:9988->9988/tcp",
		},
		{
			ports: []container.PortSummary{
				{
					PrivatePort: 9988,
					Type:        "udp",
				}, {
					PrivatePort: 9988,
					Type:        "udp",
				},
			},
			expected: "9988/udp, 9988/udp",
		},
		{
			ports: []container.PortSummary{
				{
					IP:          netip.MustParseAddr("1.2.3.4"),
					PublicPort:  9998,
					PrivatePort: 9998,
					Type:        "udp",
				}, {
					IP:          netip.MustParseAddr("1.2.3.4"),
					PublicPort:  9999,
					PrivatePort: 9999,
					Type:        "udp",
				},
			},
			expected: "1.2.3.4:9998-9999->9998-9999/udp",
		},
		{
			ports: []container.PortSummary{
				{
					IP:          netip.MustParseAddr("::1"),
					PublicPort:  9998,
					PrivatePort: 9998,
					Type:        "udp",
				}, {
					IP:          netip.MustParseAddr("::1"),
					PublicPort:  9999,
					PrivatePort: 9999,
					Type:        "udp",
				},
			},
			expected: "[::1]:9998-9999->9998-9999/udp",
		},
		{
			ports: []container.PortSummary{
				{
					IP:          netip.MustParseAddr("1.2.3.4"),
					PublicPort:  8887,
					PrivatePort: 9998,
					Type:        "udp",
				}, {
					IP:          netip.MustParseAddr("1.2.3.4"),
					PublicPort:  8888,
					PrivatePort: 9999,
					Type:        "udp",
				},
			},
			expected: "1.2.3.4:8887->9998/udp, 1.2.3.4:8888->9999/udp",
		},
		{
			ports: []container.PortSummary{
				{
					IP:          netip.MustParseAddr("::1"),
					PublicPort:  8887,
					PrivatePort: 9998,
					Type:        "udp",
				}, {
					IP:          netip.MustParseAddr("::1"),
					PublicPort:  8888,
					PrivatePort: 9999,
					Type:        "udp",
				},
			},
			expected: "[::1]:8887->9998/udp, [::1]:8888->9999/udp",
		},
		{
			ports: []container.PortSummary{
				{
					PrivatePort: 9998,
					Type:        "udp",
				}, {
					PrivatePort: 9999,
					Type:        "udp",
				},
			},
			expected: "9998-9999/udp",
		},
		{
			ports: []container.PortSummary{
				{
					IP:          netip.MustParseAddr("1.2.3.4"),
					PrivatePort: 6677,
					PublicPort:  7766,
					Type:        "tcp",
				}, {
					PrivatePort: 9988,
					PublicPort:  8899,
					Type:        "udp",
				},
			},
			expected: "9988/udp, 1.2.3.4:7766->6677/tcp",
		},
		{
			ports: []container.PortSummary{
				{
					IP:          netip.MustParseAddr("1.2.3.4"),
					PrivatePort: 9988,
					PublicPort:  8899,
					Type:        "udp",
				}, {
					IP:          netip.MustParseAddr("1.2.3.4"),
					PrivatePort: 9988,
					PublicPort:  8899,
					Type:        "tcp",
				}, {
					IP:          netip.MustParseAddr("4.3.2.1"),
					PrivatePort: 2233,
					PublicPort:  3322,
					Type:        "tcp",
				}, {
					IP:          netip.MustParseAddr("::1"),
					PrivatePort: 2233,
					PublicPort:  3322,
					Type:        "tcp",
				},
			},
			expected: "4.3.2.1:3322->2233/tcp, [::1]:3322->2233/tcp, 1.2.3.4:8899->9988/tcp, 1.2.3.4:8899->9988/udp",
		},
		{
			ports: []container.PortSummary{
				{
					PrivatePort: 9988,
					PublicPort:  8899,
					Type:        "udp",
				}, {
					IP:          netip.MustParseAddr("1.2.3.4"),
					PrivatePort: 6677,
					PublicPort:  7766,
					Type:        "tcp",
				}, {
					IP:          netip.MustParseAddr("4.3.2.1"),
					PrivatePort: 2233,
					PublicPort:  3322,
					Type:        "tcp",
				},
			},
			expected: "9988/udp, 4.3.2.1:3322->2233/tcp, 1.2.3.4:7766->6677/tcp",
		},
		{
			ports: []container.PortSummary{
				{
					PrivatePort: 80,
					Type:        "tcp",
				}, {
					PrivatePort: 1024,
					Type:        "tcp",
				}, {
					PrivatePort: 80,
					Type:        "udp",
				}, {
					PrivatePort: 1024,
					Type:        "udp",
				}, {
					IP:          netip.MustParseAddr("1.1.1.1"),
					PublicPort:  80,
					PrivatePort: 1024,
					Type:        "tcp",
				}, {
					IP:          netip.MustParseAddr("1.1.1.1"),
					PublicPort:  80,
					PrivatePort: 1024,
					Type:        "udp",
				}, {
					IP:          netip.MustParseAddr("1.1.1.1"),
					PublicPort:  1024,
					PrivatePort: 80,
					Type:        "tcp",
				}, {
					IP:          netip.MustParseAddr("1.1.1.1"),
					PublicPort:  1024,
					PrivatePort: 80,
					Type:        "udp",
				}, {
					IP:          netip.MustParseAddr("2.1.1.1"),
					PublicPort:  80,
					PrivatePort: 1024,
					Type:        "tcp",
				}, {
					IP:          netip.MustParseAddr("2.1.1.1"),
					PublicPort:  80,
					PrivatePort: 1024,
					Type:        "udp",
				}, {
					IP:          netip.MustParseAddr("2.1.1.1"),
					PublicPort:  1024,
					PrivatePort: 80,
					Type:        "tcp",
				}, {
					IP:          netip.MustParseAddr("2.1.1.1"),
					PublicPort:  1024,
					PrivatePort: 80,
					Type:        "udp",
				}, {
					PrivatePort: 12345,
					Type:        "sctp",
				},
			},
			expected: "80/tcp, 80/udp, 1024/tcp, 1024/udp, 12345/sctp, 1.1.1.1:1024->80/tcp, 1.1.1.1:1024->80/udp, 2.1.1.1:1024->80/tcp, 2.1.1.1:1024->80/udp, 1.1.1.1:80->1024/tcp, 1.1.1.1:80->1024/udp, 2.1.1.1:80->1024/tcp, 2.1.1.1:80->1024/udp", //nolint:revive // ignore line-length-limit (revive)
		},
	}

	for _, port := range cases {
		actual := DisplayablePorts(port.ports)
		assert.Check(t, is.Equal(port.expected, actual))
	}
}
