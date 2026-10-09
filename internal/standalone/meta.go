//go:build linux

package standalone

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/containerd/containerd/v2/core/containers"
	cerrdefs "github.com/containerd/errdefs"
	"github.com/containerd/typeurl/v2"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
)

const (
	// metaExtension is the key under which the Docker-level container
	// metadata is stored in the containerd container record.
	metaExtension = "io.docker.standalone/container"
	// labelName indexes containers by their Docker name.
	labelName = "io.docker.standalone.name"
	// labelHostname is set on the containerd container so that "docker ps"
	// style lookups do not need to decode the extension.
	labelImage = "io.docker.standalone.image"

	metaTypeURL = "github.com/docker/cli/internal/standalone/containerMeta"
)

func init() {
	typeurl.Register(&containerMeta{}, metaTypeURL)
}

// containerMeta is the Docker-level view of a container. It is stored as an
// extension on the containerd container record, next to the OCI spec.
type containerMeta struct {
	Name             string                    `json:"name"`
	Created          time.Time                 `json:"created"`
	Config           *container.Config         `json:"config"`
	HostConfig       *container.HostConfig     `json:"hostConfig"`
	NetworkingConfig *network.NetworkingConfig `json:"networkingConfig,omitempty"`
	Platform         string                    `json:"platform"`
	ImageID          string                    `json:"imageID"`
	ImageRef         string                    `json:"imageRef"`
	Path             string                    `json:"path"`
	Args             []string                  `json:"args"`
	Mounts           []container.MountPoint    `json:"mounts"`
	AnonymousVolumes []string                  `json:"anonymousVolumes,omitempty"`
	LogPath          string                    `json:"logPath"`
	HostnamePath     string                    `json:"hostnamePath"`
	HostsPath        string                    `json:"hostsPath"`
	ResolvConfPath   string                    `json:"resolvConfPath"`
	Runtime          string                    `json:"runtime"`
	State            containerState            `json:"state"`
	Network          networkState              `json:"network"`
}

// containerState is the last observed runtime state of the container. It is
// updated by whichever process observes a transition (the CLI on start/stop,
// the logging helper on exit).
type containerState struct {
	Status       string    `json:"status"` // created, running, paused, exited
	Pid          int       `json:"pid"`
	ExitCode     int       `json:"exitCode"`
	Error        string    `json:"error,omitempty"`
	StartedAt    time.Time `json:"startedAt"`
	FinishedAt   time.Time `json:"finishedAt"`
	RestartCount int       `json:"restartCount"`
	OOMKilled    bool      `json:"oomKilled"`
}

// networkState records how the container was networked.
type networkState struct {
	Backend    string                `json:"backend"` // bridge, pasta, slirp4netns, host, none
	NetnsPath  string                `json:"netnsPath,omitempty"`
	IPAddress  string                `json:"ipAddress,omitempty"`
	IPPrefix   int                   `json:"ipPrefix,omitempty"`
	Gateway    string                `json:"gateway,omitempty"`
	MacAddress string                `json:"macAddress,omitempty"`
	DNS        []string              `json:"dns,omitempty"`
	Ports      []network.PortBinding `json:"ports,omitempty"`
	PortMap    network.PortMap       `json:"portMap,omitempty"`
}

func marshalMeta(m *containerMeta) (typeurl.Any, error) {
	return typeurl.MarshalAny(m)
}

func unmarshalMeta(c containers.Container) (*containerMeta, error) {
	a, ok := c.Extensions[metaExtension]
	if !ok {
		return nil, fmt.Errorf("container %s has no standalone metadata: %w", c.ID, cerrdefs.ErrNotFound)
	}
	v, err := typeurl.UnmarshalAny(a)
	if err != nil {
		return nil, fmt.Errorf("decoding metadata for container %s: %w", c.ID, err)
	}
	m, ok := v.(*containerMeta)
	if !ok {
		return nil, fmt.Errorf("unexpected metadata type %T for container %s", v, c.ID)
	}
	return m, nil
}

// updateMeta stores the metadata on the containerd record and persists it.
func updateMeta(ctx context.Context, store containers.Store, c containers.Container, m *containerMeta) error {
	a, err := marshalMeta(m)
	if err != nil {
		return err
	}
	if c.Extensions == nil {
		c.Extensions = map[string]typeurl.Any{}
	}
	c.Extensions[metaExtension] = a
	if c.Labels == nil {
		c.Labels = map[string]string{}
	}
	c.Labels[labelName] = m.Name
	_, err = store.Update(ctx, c, "extensions."+metaExtension, "labels."+labelName)
	return err
}

// loadContainer resolves a container by full ID, ID prefix or name.
func loadContainer(ctx context.Context, store containers.Store, ref string) (containers.Container, *containerMeta, error) {
	ref = strings.TrimPrefix(ref, "/")
	if ref == "" {
		return containers.Container{}, nil, fmt.Errorf("empty container reference: %w", cerrdefs.ErrInvalidArgument)
	}
	// Exact ID.
	if c, err := store.Get(ctx, ref); err == nil {
		m, err := unmarshalMeta(c)
		return c, m, err
	}
	all, err := store.List(ctx)
	if err != nil {
		return containers.Container{}, nil, err
	}
	var (
		byName   []containers.Container
		byPrefix []containers.Container
	)
	for _, c := range all {
		if c.Labels[labelName] == ref {
			byName = append(byName, c)
		} else if strings.HasPrefix(c.ID, ref) {
			byPrefix = append(byPrefix, c)
		}
	}
	switch {
	case len(byName) == 1:
		m, err := unmarshalMeta(byName[0])
		return byName[0], m, err
	case len(byPrefix) == 1:
		m, err := unmarshalMeta(byPrefix[0])
		return byPrefix[0], m, err
	case len(byPrefix) > 1:
		return containers.Container{}, nil, fmt.Errorf("multiple containers found with ID prefix %q: %w", ref, cerrdefs.ErrInvalidArgument)
	}
	return containers.Container{}, nil, fmt.Errorf("no such container: %s: %w", ref, cerrdefs.ErrNotFound)
}

func newID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
