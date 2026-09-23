//go:build linux

package standalone

import (
	"testing"
	"time"

	"github.com/containerd/containerd/v2/core/containers"
	cerrdefs "github.com/containerd/errdefs"
	"github.com/containerd/typeurl/v2"
	"github.com/moby/moby/api/types/container"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestMetaRoundTrip(t *testing.T) {
	created := time.Now().UTC().Truncate(time.Second)
	m := &containerMeta{
		Name:       "web",
		Created:    created,
		Config:     &container.Config{Image: "alpine", Hostname: "abc"},
		HostConfig: &container.HostConfig{NetworkMode: "bridge"},
		ImageID:    "sha256:deadbeef",
		Path:       "/bin/sh",
		Args:       []string{"-c", "echo hi"},
		State:      containerState{Status: statusRunning, Pid: 42},
		Network:    networkState{Backend: networkPasta, IPAddress: "10.0.2.100"},
	}
	stored, err := marshalMeta(m)
	assert.NilError(t, err)

	ctr := containers.Container{ID: "id", Extensions: map[string]typeurl.Any{metaExtension: stored}}
	got, err := unmarshalMeta(ctr)
	assert.NilError(t, err)
	assert.Check(t, is.Equal(got.Name, "web"))
	assert.Check(t, is.Equal(got.Config.Image, "alpine"))
	assert.Check(t, is.Equal(got.State.Status, statusRunning))
	assert.Check(t, is.Equal(got.State.Pid, 42))
	assert.Check(t, is.Equal(got.Network.IPAddress, "10.0.2.100"))
	assert.Check(t, got.Created.Equal(created))
}

func TestUnmarshalMetaMissing(t *testing.T) {
	_, err := unmarshalMeta(containers.Container{ID: "id"})
	assert.Check(t, cerrdefs.IsNotFound(err))
}

func TestNewID(t *testing.T) {
	a, b := newID(), newID()
	assert.Check(t, is.Equal(len(a), 64))
	assert.Check(t, a != b)
}

func TestStatusString(t *testing.T) {
	e := &engine{cfg: &config{}}
	now := time.Now()

	assert.Check(t, is.Equal(e.statusString(&containerMeta{State: containerState{Status: statusCreated}}), "Created"))
	assert.Check(t, is.Contains(e.statusString(&containerMeta{State: containerState{Status: statusRunning, StartedAt: now}}), "Up "))
	assert.Check(t, is.Contains(e.statusString(&containerMeta{State: containerState{Status: statusPaused, StartedAt: now}}), "(Paused)"))
	assert.Check(t, is.Contains(e.statusString(&containerMeta{State: containerState{Status: statusExited, ExitCode: 3, FinishedAt: now}}), "Exited (3) "))
	// A container whose exit time is unknown must not be reported as having
	// exited centuries ago.
	assert.Check(t, is.Equal(e.statusString(&containerMeta{State: containerState{Status: statusExited, ExitCode: 1}}), "Exited (1)"))
}

func TestFormatTime(t *testing.T) {
	assert.Check(t, is.Equal(formatTime(time.Time{}), "0001-01-01T00:00:00Z"))
	ts := time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC)
	assert.Check(t, is.Equal(formatTime(ts), "2024-05-06T07:08:09Z"))
}

func TestRandomNameFormat(t *testing.T) {
	assert.Check(t, len(nameAdjectives) > 50)
	assert.Check(t, len(nameSurnames) > 50)
	for _, s := range append(append([]string{}, nameAdjectives...), nameSurnames...) {
		assert.NilError(t, validateName(s+"_x"))
	}
}
