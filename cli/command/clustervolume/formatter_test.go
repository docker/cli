package clustervolume

import (
	"gotest.tools/v3/assert"
	"testing"

	"bytes"
	"encoding/json"
	"strings"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/swarm"
)

// TestInspectFormatWriteJSON is a very, very simple tests that writes the JSON
// formatted output and reads it back out. All this does is ensure that the
// formatter works, the code compiles, etc. Nothing to get excited about, but
// necessary along the path to development.
func TestInspectFormatWriteJSON(t *testing.T) {
	var out bytes.Buffer

	ctx := formatter.Context{
		Output: &out,
		Format: NewFormat("{{json .}}"),
	}
	v := swarm.Volume{
		ID: "volume1",
		Spec: swarm.VolumeSpec{
			Annotations: swarm.Annotations{
				Name: "volume1Name",
			},
			Group: "volume1Group",
			Driver: &mount.Driver{
				Name: "volume1Driver",
			},
			AccessMode: &swarm.VolumeAccessMode{
				Scope:   swarm.VolumeScopeSingleNode,
				Sharing: swarm.VolumeSharingNone,
			},
			Secrets: []swarm.VolumeSecret{
				{Key: "key1", Secret: "secret1"},
				{Key: "key2", Secret: "secret2"},
			},
			AccessibilityRequirements: &swarm.TopologyRequirement{
				Requisite: []swarm.Topology{
					{Segments: map[string]string{"foo": "bar"}},
				},
				Preferred: []swarm.Topology{
					{Segments: map[string]string{"baz": "bat"}},
				},
			},
			CapacityRange: &swarm.CapacityRange{
				RequiredBytes: 8,
				LimitBytes:    8000000,
			},
			Availability: swarm.VolumeAvailabilityActive,
		},
	}

	vBytes, err := json.Marshal(v)
	assert.NilError(t, err)

	getRef := func(ref string) (interface{}, []byte, error) {
		return v, vBytes, err
	}

	err = InspectFormatWrite(ctx, []string{"volume1"}, getRef)
	assert.NilError(t, err)
	// the output from the formatter adds a newline
	trimmed := strings.TrimSpace(out.String())
	assert.Equal(t, string(vBytes), trimmed)
}
