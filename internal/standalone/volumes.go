//go:build linux

package standalone

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/api/types/volume"
	"github.com/moby/moby/client"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// Volumes are implemented by the "local" driver only: a directory per volume
// under <root>/volumes/<name>/_data with a small JSON metadata file.

type volumeMeta struct {
	Name      string            `json:"name"`
	CreatedAt time.Time         `json:"createdAt"`
	Labels    map[string]string `json:"labels,omitempty"`
	Anonymous bool              `json:"anonymous,omitempty"`
}

func (e *engine) volumePath(name string) string {
	return filepath.Join(e.cfg.volumesDir(), name)
}

func (e *engine) volumeDataPath(name string) string {
	return filepath.Join(e.volumePath(name), "_data")
}

func validVolumeName(name string) bool {
	if name == "" || len(name) > 255 {
		return false
	}
	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' && r != '.' && r != '-' {
			return false
		}
	}
	return true
}

// createVolume creates a local volume if it does not exist and returns its
// metadata.
func (e *engine) createVolume(name string, labels map[string]string, anonymous bool) (*volumeMeta, error) {
	if name == "" {
		name = newID()
		anonymous = true
	}
	if !validVolumeName(name) {
		return nil, fmt.Errorf("invalid volume name %q: %w", name, cerrdefs.ErrInvalidArgument)
	}
	if m, err := e.getVolume(name); err == nil {
		return m, nil
	}
	if err := os.MkdirAll(e.volumeDataPath(name), 0o755); err != nil {
		return nil, err
	}
	m := &volumeMeta{Name: name, CreatedAt: time.Now().UTC(), Labels: labels, Anonymous: anonymous}
	data, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(e.volumePath(name), "volume.json"), data, 0o600); err != nil {
		return nil, err
	}
	return m, nil
}

func (e *engine) getVolume(name string) (*volumeMeta, error) {
	data, err := os.ReadFile(filepath.Join(e.volumePath(name), "volume.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("no such volume: %s: %w", name, cerrdefs.ErrNotFound)
		}
		return nil, err
	}
	var m volumeMeta
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (e *engine) listVolumes() ([]*volumeMeta, error) {
	entries, err := os.ReadDir(e.cfg.volumesDir())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var out []*volumeMeta
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		m, err := e.getVolume(ent.Name())
		if err != nil {
			continue
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (e *engine) removeVolume(name string) error {
	if _, err := e.getVolume(name); err != nil {
		return err
	}
	return os.RemoveAll(e.volumePath(name))
}

func (e *engine) volumeToAPI(m *volumeMeta) volume.Volume {
	return volume.Volume{
		Name:       m.Name,
		Driver:     "local",
		Mountpoint: e.volumeDataPath(m.Name),
		CreatedAt:  m.CreatedAt.Format(time.RFC3339),
		Labels:     m.Labels,
		Options:    map[string]string{},
		Scope:      "local",
	}
}

// volumesInUse returns the set of volume names referenced by any container.
func volumesInUse(ctx context.Context, s *session) (map[string]string, error) {
	ctrs, err := s.containers().List(ctx)
	if err != nil {
		return nil, err
	}
	used := map[string]string{}
	for _, c := range ctrs {
		m, err := unmarshalMeta(c)
		if err != nil {
			continue
		}
		for _, mp := range m.Mounts {
			if mp.Type == mount.TypeVolume && mp.Name != "" {
				used[mp.Name] = c.ID
			}
		}
	}
	return used, nil
}

// ---- Mount resolution for container creation

// resolveMounts converts Binds, Mounts, Tmpfs and image/config volumes into
// OCI mounts and Docker mount points, creating anonymous volumes as needed.
func (e *engine) resolveMounts(cfg *container.Config, hc *container.HostConfig, imageVolumes map[string]struct{}) ([]specs.Mount, []container.MountPoint, []string, error) {
	var (
		ociMounts   []specs.Mount
		mountPoints []container.MountPoint
		anonymous   []string
		seen        = map[string]bool{}
	)
	add := func(om specs.Mount, mp container.MountPoint) {
		dest := filepath.Clean(om.Destination)
		if seen[dest] {
			return
		}
		seen[dest] = true
		om.Destination = dest
		mp.Destination = dest
		ociMounts = append(ociMounts, om)
		mountPoints = append(mountPoints, mp)
	}

	for _, b := range hc.Binds {
		om, mp, anon, err := e.resolveBind(b)
		if err != nil {
			return nil, nil, nil, err
		}
		if anon != "" {
			anonymous = append(anonymous, anon)
		}
		add(om, mp)
	}
	for _, m := range hc.Mounts {
		om, mp, anon, err := e.resolveMount(m)
		if err != nil {
			return nil, nil, nil, err
		}
		if anon != "" {
			anonymous = append(anonymous, anon)
		}
		add(om, mp)
	}
	for dest, opts := range hc.Tmpfs {
		options := []string{"nosuid", "nodev", "noexec", "rprivate"}
		if opts != "" {
			options = append(options, strings.Split(opts, ",")...)
		}
		add(specs.Mount{Destination: dest, Type: "tmpfs", Source: "tmpfs", Options: options},
			container.MountPoint{Type: mount.TypeTmpfs, Destination: dest, Mode: opts, RW: true})
	}
	volumes := map[string]struct{}{}
	for v := range imageVolumes {
		volumes[v] = struct{}{}
	}
	for v := range cfg.Volumes {
		volumes[v] = struct{}{}
	}
	dests := make([]string, 0, len(volumes))
	for d := range volumes {
		dests = append(dests, d)
	}
	sort.Strings(dests)
	for _, dest := range dests {
		if seen[filepath.Clean(dest)] {
			continue
		}
		vm, err := e.createVolume("", nil, true)
		if err != nil {
			return nil, nil, nil, err
		}
		anonymous = append(anonymous, vm.Name)
		add(bindMount(e.volumeDataPath(vm.Name), dest, false, ""), container.MountPoint{
			Type:        mount.TypeVolume,
			Name:        vm.Name,
			Source:      e.volumeDataPath(vm.Name),
			Destination: dest,
			Driver:      "local",
			Mode:        "z",
			RW:          true,
			Propagation: mount.PropagationRPrivate,
		})
	}
	return ociMounts, mountPoints, anonymous, nil
}

func bindMount(src, dest string, readOnly bool, propagation mount.Propagation) specs.Mount {
	opts := []string{"rbind"}
	if readOnly {
		opts = append(opts, "ro")
	} else {
		opts = append(opts, "rw")
	}
	if propagation == "" {
		propagation = mount.PropagationRPrivate
	}
	opts = append(opts, string(propagation))
	return specs.Mount{Destination: dest, Type: "bind", Source: src, Options: opts}
}

// resolveBind handles the "-v" syntax: [source:]destination[:options].
//
//nolint:gocyclo // parses the full -v syntax
func (e *engine) resolveBind(spec string) (specs.Mount, container.MountPoint, string, error) {
	parts := strings.Split(spec, ":")
	var src, dest, opts string
	switch len(parts) {
	case 1:
		dest = parts[0]
	case 2:
		src, dest = parts[0], parts[1]
	case 3:
		src, dest, opts = parts[0], parts[1], parts[2]
	default:
		return specs.Mount{}, container.MountPoint{}, "", fmt.Errorf("invalid volume specification %q: %w", spec, cerrdefs.ErrInvalidArgument)
	}
	if !filepath.IsAbs(dest) {
		return specs.Mount{}, container.MountPoint{}, "", fmt.Errorf("invalid volume specification %q: destination must be absolute: %w", spec, cerrdefs.ErrInvalidArgument)
	}
	readOnly := false
	var propagation mount.Propagation
	for _, o := range strings.Split(opts, ",") {
		switch o {
		case "ro":
			readOnly = true
		case "rw", "", "z", "Z", "nocopy":
		case "rprivate", "private", "rshared", "shared", "rslave", "slave":
			propagation = mount.Propagation(o)
		default:
			return specs.Mount{}, container.MountPoint{}, "", fmt.Errorf("invalid volume option %q: %w", o, cerrdefs.ErrInvalidArgument)
		}
	}
	if src == "" || (!filepath.IsAbs(src) && !strings.HasPrefix(src, ".") && !strings.HasPrefix(src, "~")) {
		// Named or anonymous volume.
		vm, err := e.createVolume(src, nil, src == "")
		if err != nil {
			return specs.Mount{}, container.MountPoint{}, "", err
		}
		anon := ""
		if src == "" {
			anon = vm.Name
		}
		return bindMount(e.volumeDataPath(vm.Name), dest, readOnly, propagation), container.MountPoint{
			Type:        mount.TypeVolume,
			Name:        vm.Name,
			Source:      e.volumeDataPath(vm.Name),
			Destination: dest,
			Driver:      "local",
			Mode:        opts,
			RW:          !readOnly,
			Propagation: mount.PropagationRPrivate,
		}, anon, nil
	}
	src, err := filepath.Abs(src)
	if err != nil {
		return specs.Mount{}, container.MountPoint{}, "", err
	}
	if _, err := os.Stat(src); errors.Is(err, os.ErrNotExist) {
		// Docker creates missing bind source directories.
		if err := os.MkdirAll(src, 0o755); err != nil {
			return specs.Mount{}, container.MountPoint{}, "", err
		}
	}
	if propagation == "" {
		propagation = mount.PropagationRPrivate
	}
	return bindMount(src, dest, readOnly, propagation),
		container.MountPoint{Type: mount.TypeBind, Source: src, Destination: dest, Mode: opts, RW: !readOnly, Propagation: propagation}, "", nil
}

// resolveMount handles the "--mount" syntax.
//
//nolint:gocyclo // one branch per supported mount type and its options
func (e *engine) resolveMount(m mount.Mount) (specs.Mount, container.MountPoint, string, error) {
	if !filepath.IsAbs(m.Target) {
		return specs.Mount{}, container.MountPoint{}, "", fmt.Errorf("invalid mount target %q: must be absolute: %w", m.Target, cerrdefs.ErrInvalidArgument)
	}
	switch m.Type { //nolint:exhaustive // the unsupported types are rejected by the default case
	case mount.TypeBind:
		var propagation mount.Propagation
		if m.BindOptions != nil {
			propagation = m.BindOptions.Propagation
		}
		if _, err := os.Stat(m.Source); err != nil {
			if !errors.Is(err, os.ErrNotExist) || m.BindOptions == nil || !m.BindOptions.CreateMountpoint {
				return specs.Mount{}, container.MountPoint{}, "", fmt.Errorf("bind source path does not exist: %s: %w", m.Source, cerrdefs.ErrInvalidArgument)
			}
			if err := os.MkdirAll(m.Source, 0o755); err != nil {
				return specs.Mount{}, container.MountPoint{}, "", err
			}
		}
		if propagation == "" {
			propagation = mount.PropagationRPrivate
		}
		return bindMount(m.Source, m.Target, m.ReadOnly, propagation),
			container.MountPoint{Type: mount.TypeBind, Source: m.Source, Destination: m.Target, RW: !m.ReadOnly, Propagation: propagation}, "", nil
	case mount.TypeVolume:
		var labels map[string]string
		if m.VolumeOptions != nil {
			labels = m.VolumeOptions.Labels
		}
		vm, err := e.createVolume(m.Source, labels, m.Source == "")
		if err != nil {
			return specs.Mount{}, container.MountPoint{}, "", err
		}
		anon := ""
		if m.Source == "" {
			anon = vm.Name
		}
		return bindMount(e.volumeDataPath(vm.Name), m.Target, m.ReadOnly, ""), container.MountPoint{
			Type:        mount.TypeVolume,
			Name:        vm.Name,
			Source:      e.volumeDataPath(vm.Name),
			Destination: m.Target,
			Driver:      "local",
			RW:          !m.ReadOnly,
			Propagation: mount.PropagationRPrivate,
		}, anon, nil
	case mount.TypeTmpfs:
		options := []string{"nosuid", "nodev", "noexec", "rprivate"}
		if m.ReadOnly {
			options = append(options, "ro")
		}
		if m.TmpfsOptions != nil {
			if m.TmpfsOptions.SizeBytes > 0 {
				options = append(options, fmt.Sprintf("size=%d", m.TmpfsOptions.SizeBytes))
			}
			if m.TmpfsOptions.Mode != 0 {
				options = append(options, fmt.Sprintf("mode=%o", m.TmpfsOptions.Mode))
			}
		}
		return specs.Mount{Destination: m.Target, Type: "tmpfs", Source: "tmpfs", Options: options},
			container.MountPoint{Type: mount.TypeTmpfs, Destination: m.Target, RW: !m.ReadOnly}, "", nil
	default:
		return specs.Mount{}, container.MountPoint{}, "", fmt.Errorf("unsupported mount type %q: %w", m.Type, cerrdefs.ErrInvalidArgument)
	}
}

// ---- Volume API

func (c *apiClient) VolumeCreate(_ context.Context, options client.VolumeCreateOptions) (client.VolumeCreateResult, error) {
	if options.Driver != "" && options.Driver != "local" {
		return client.VolumeCreateResult{}, fmt.Errorf("volume driver %q is not supported (only local): %w", options.Driver, cerrdefs.ErrNotImplemented)
	}
	m, err := c.eng.createVolume(options.Name, options.Labels, false)
	if err != nil {
		return client.VolumeCreateResult{}, err
	}
	return client.VolumeCreateResult{Volume: c.eng.volumeToAPI(m)}, nil
}

func (c *apiClient) VolumeInspect(_ context.Context, name string, _ client.VolumeInspectOptions) (client.VolumeInspectResult, error) {
	m, err := c.eng.getVolume(name)
	if err != nil {
		return client.VolumeInspectResult{}, err
	}
	return client.VolumeInspectResult{Volume: c.eng.volumeToAPI(m)}, nil
}

func (c *apiClient) VolumeList(ctx context.Context, options client.VolumeListOptions) (client.VolumeListResult, error) {
	vols, err := c.eng.listVolumes()
	if err != nil {
		return client.VolumeListResult{}, err
	}
	var used map[string]string
	if _, ok := options.Filters["dangling"]; ok {
		if err := c.eng.withSession(ctx, func(ctx context.Context, s *session) error {
			used, err = volumesInUse(ctx, s)
			return err
		}); err != nil {
			return client.VolumeListResult{}, err
		}
	}
	res := client.VolumeListResult{Items: []volume.Volume{}}
	for _, m := range vols {
		if !matchVolumeFilters(options.Filters, m, used) {
			continue
		}
		res.Items = append(res.Items, c.eng.volumeToAPI(m))
	}
	return res, nil
}

//nolint:gocyclo // one branch per supported filter
func matchVolumeFilters(f client.Filters, m *volumeMeta, used map[string]string) bool {
	for key, values := range f {
		switch key {
		case "name":
			ok := false
			for v := range values {
				if strings.Contains(m.Name, v) {
					ok = true
				}
			}
			if !ok {
				return false
			}
		case "driver":
			if _, ok := values["local"]; !ok {
				return false
			}
		case "label":
			for v := range values {
				k, val, hasVal := strings.Cut(v, "=")
				lv, ok := m.Labels[k]
				if !ok || (hasVal && lv != val) {
					return false
				}
			}
		case "dangling":
			wantDangling := true
			for v := range values {
				if v == "false" || v == "0" {
					wantDangling = false
				}
			}
			_, inUse := used[m.Name]
			if wantDangling == inUse {
				return false
			}
		}
	}
	return true
}

func (c *apiClient) VolumeRemove(ctx context.Context, name string, options client.VolumeRemoveOptions) (client.VolumeRemoveResult, error) {
	err := c.eng.withSession(ctx, func(ctx context.Context, s *session) error {
		used, err := volumesInUse(ctx, s)
		if err != nil {
			return err
		}
		if ctr, ok := used[name]; ok && !options.Force {
			return fmt.Errorf("volume %s is in use by container %s: %w", name, shortID(ctr), cerrdefs.ErrConflict)
		}
		return c.eng.removeVolume(name)
	})
	return client.VolumeRemoveResult{}, err
}

func (c *apiClient) VolumePrune(ctx context.Context, options client.VolumePruneOptions) (client.VolumePruneResult, error) {
	var res client.VolumePruneResult
	err := c.eng.withSession(ctx, func(ctx context.Context, s *session) error {
		used, err := volumesInUse(ctx, s)
		if err != nil {
			return err
		}
		vols, err := c.eng.listVolumes()
		if err != nil {
			return err
		}
		all := false
		for v := range options.Filters["all"] {
			if v == "true" || v == "1" {
				all = true
			}
		}
		for _, m := range vols {
			if _, inUse := used[m.Name]; inUse {
				continue
			}
			if !m.Anonymous && !all {
				continue
			}
			size := dirSize(c.eng.volumeDataPath(m.Name))
			if err := c.eng.removeVolume(m.Name); err != nil {
				return err
			}
			res.Report.VolumesDeleted = append(res.Report.VolumesDeleted, m.Name)
			res.Report.SpaceReclaimed += uint64(size)
		}
		return nil
	})
	return res, err
}

func dirSize(path string) int64 {
	var size int64
	_ = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}
