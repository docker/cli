//go:build linux

package standalone

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/containerd/containerd/v2/core/content"
	"github.com/containerd/containerd/v2/core/images"
	cerrdefs "github.com/containerd/errdefs"
	"github.com/containerd/platforms"
	"github.com/distribution/reference"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/client"
	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/identity"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// resolvedImage bundles an image record with its platform-specific config.
type resolvedImage struct {
	img        images.Image
	configDesc ocispec.Descriptor
	config     dockerspec.DockerOCIImage
	platform   platforms.MatchComparer
}

// ID returns the Docker image ID, which is the digest of the image config.
func (r *resolvedImage) ID() string {
	return r.configDesc.Digest.String()
}

// resolveImage finds an image by reference or (truncated) ID and reads its
// config for the given platform.
//
//nolint:gocyclo // resolves references, digests and truncated image IDs
func resolveImage(ctx context.Context, s *session, ref string, platform platforms.MatchComparer) (*resolvedImage, error) {
	if platform == nil {
		platform = platforms.Default()
	}
	store := s.images()
	var img images.Image
	named, nerr := normalizeImageRef(ref)
	if nerr == nil {
		i, err := store.Get(ctx, named.String())
		if err == nil {
			img = i
		} else if !cerrdefs.IsNotFound(err) {
			return nil, err
		}
	}
	if img.Name == "" && isImageID(ref) {
		all, err := store.List(ctx)
		if err != nil {
			return nil, err
		}
		var matches []images.Image
		for _, i := range all {
			desc, err := i.Config(ctx, s.content(), platform)
			if err != nil {
				continue
			}
			if matchesImageID(desc.Digest, ref) || matchesImageID(i.Target.Digest, ref) {
				matches = append(matches, i)
			}
		}
		if len(matches) > 0 {
			// Multiple names may point at the same image; any of them will do
			// as long as the config digest is unique.
			ids := map[digest.Digest]struct{}{}
			for _, m := range matches {
				desc, _ := m.Config(ctx, s.content(), platform)
				ids[desc.Digest] = struct{}{}
			}
			if len(ids) > 1 {
				return nil, fmt.Errorf("multiple images found with ID prefix %q: %w", ref, cerrdefs.ErrInvalidArgument)
			}
			img = matches[0]
		}
	}
	if img.Name == "" {
		if nerr != nil {
			return nil, nerr
		}
		return nil, fmt.Errorf("no such image: %s: %w", ref, cerrdefs.ErrNotFound)
	}
	return readImage(ctx, s, img, platform)
}

func readImage(ctx context.Context, s *session, img images.Image, platform platforms.MatchComparer) (*resolvedImage, error) {
	if platform == nil {
		platform = platforms.Default()
	}
	desc, err := img.Config(ctx, s.content(), platform)
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return nil, fmt.Errorf("image %s is not available for platform %s: %w", familiarizeImageRef(img.Name), platforms.DefaultString(), cerrdefs.ErrNotFound)
		}
		return nil, err
	}
	blob, err := content.ReadBlob(ctx, s.content(), desc)
	if err != nil {
		return nil, fmt.Errorf("reading image config: %w", err)
	}
	var cfg dockerspec.DockerOCIImage
	if err := json.Unmarshal(blob, &cfg); err != nil {
		return nil, fmt.Errorf("decoding image config: %w", err)
	}
	return &resolvedImage{img: img, configDesc: desc, config: cfg, platform: platform}, nil
}

// ---- API client methods

func (c *apiClient) ImageList(ctx context.Context, options client.ImageListOptions) (client.ImageListResult, error) {
	var res client.ImageListResult
	err := c.eng.withSession(ctx, func(ctx context.Context, s *session) error {
		all, err := s.images().List(ctx)
		if err != nil {
			return err
		}
		byID := map[string]*image.Summary{}
		var order []string
		for _, img := range all {
			ri, err := readImage(ctx, s, img, nil)
			if err != nil {
				// Image content for this platform may not be present
				// (e.g. a foreign-platform image loaded from an archive).
				continue
			}
			if !matchImageFilters(options.Filters, ri) {
				continue
			}
			id := ri.ID()
			sum, ok := byID[id]
			if !ok {
				size, _ := img.Size(ctx, s.content(), ri.platform)
				var created int64
				if ri.config.Created != nil {
					created = ri.config.Created.Unix()
				}
				sum = &image.Summary{
					ID:          id,
					Created:     created,
					Size:        size,
					SharedSize:  -1,
					Containers:  -1,
					Labels:      ri.config.Config.Labels,
					RepoTags:    []string{},
					RepoDigests: []string{},
				}
				if options.Manifests {
					sum.Manifests, sum.Size = manifestSummaries(ctx, s, img, ri)
					desc := img.Target
					sum.Descriptor = &desc
				}
				byID[id] = sum
				order = append(order, id)
			}
			named, err := reference.ParseNamed(img.Name)
			if err != nil {
				continue
			}
			if tagged, ok := named.(reference.Tagged); ok {
				sum.RepoTags = append(sum.RepoTags, reference.FamiliarName(named)+":"+tagged.Tag())
			}
			sum.RepoDigests = append(sum.RepoDigests, reference.FamiliarName(named)+"@"+img.Target.Digest.String())
		}
		res.Items = make([]image.Summary, 0, len(order))
		for _, id := range order {
			sum := byID[id]
			sort.Strings(sum.RepoTags)
			sort.Strings(sum.RepoDigests)
			res.Items = append(res.Items, *sum)
		}
		sort.Slice(res.Items, func(i, j int) bool { return res.Items[i].Created > res.Items[j].Created })
		return nil
	})
	return res, err
}

// manifestSummaries describes the per-platform manifests of an image (used
// by the tree view of "docker images") and returns the total disk usage.
//
//nolint:gocyclo // walks every manifest of a possibly multi-platform image
func manifestSummaries(ctx context.Context, s *session, img images.Image, ri *resolvedImage) ([]image.ManifestSummary, int64) {
	cs := s.content()
	var (
		out   []image.ManifestSummary
		total int64
	)
	ctrs, _ := s.containers().List(ctx)
	manifests, err := images.Children(ctx, cs, img.Target)
	if err != nil || !images.IsIndexType(img.Target.MediaType) {
		manifests = []ocispec.Descriptor{img.Target}
	}
	for _, mdesc := range manifests {
		if !images.IsManifestType(mdesc.MediaType) {
			continue
		}
		ms := image.ManifestSummary{ID: mdesc.Digest.String(), Descriptor: mdesc, Kind: image.ManifestKindImage}
		if mdesc.Annotations["vnd.docker.reference.type"] == "attestation-manifest" {
			ms.Kind = image.ManifestKindAttestation
			ms.AttestationData = &image.AttestationProperties{For: digest.Digest(mdesc.Annotations["vnd.docker.reference.digest"])}
		}
		matcher := platforms.All
		if mdesc.Platform != nil {
			matcher = platforms.Only(*mdesc.Platform)
		}
		available, _, present, _, err := images.Check(ctx, cs, mdesc, matcher)
		if err == nil {
			ms.Available = available
			for _, p := range present {
				ms.Size.Content += p.Size
			}
		}
		ms.Size.Total = ms.Size.Content
		if ms.Kind == image.ManifestKindImage {
			var platform ocispec.Platform
			if mdesc.Platform != nil {
				platform = *mdesc.Platform
			} else {
				platform = platforms.MustParse(platforms.Format(ocispec.Platform{OS: ri.config.OS, Architecture: ri.config.Architecture, Variant: ri.config.Variant}))
			}
			ms.ImageData = &image.ImageProperties{Platform: platform, Containers: []string{}}
			if available {
				if diffIDs, err := images.RootFS(ctx, cs, mustConfig(ctx, cs, mdesc, matcher)); err == nil && len(diffIDs) > 0 {
					if u, err := s.snapshotter().Usage(ctx, identity.ChainID(diffIDs).String()); err == nil {
						ms.ImageData.Size.Unpacked = u.Size
						ms.Size.Total += u.Size
					}
				}
			}
			for _, c := range ctrs {
				if c.Image == img.Name {
					ms.ImageData.Containers = append(ms.ImageData.Containers, c.ID)
				}
			}
		}
		total += ms.Size.Total
		out = append(out, ms)
	}
	return out, total
}

func mustConfig(ctx context.Context, cs content.Store, desc ocispec.Descriptor, matcher platforms.MatchComparer) ocispec.Descriptor {
	c, err := images.Config(ctx, cs, desc, matcher)
	if err != nil {
		return ocispec.Descriptor{}
	}
	return c
}

func matchImageFilters(f client.Filters, ri *resolvedImage) bool {
	for key, values := range f {
		switch key {
		case "reference":
			matched := false
			for v := range values {
				if ok, _ := reference.FamiliarMatch(v, mustParseNamed(ri.img.Name)); ok {
					matched = true
				}
			}
			if !matched {
				return false
			}
		case "label":
			for v := range values {
				k, val, hasVal := strings.Cut(v, "=")
				lv, ok := ri.config.Config.Labels[k]
				if !ok || (hasVal && lv != val) {
					return false
				}
			}
		case "dangling":
			// Images are always tagged in this backend.
			for v := range values {
				if v == "true" {
					return false
				}
			}
		}
	}
	return true
}

func mustParseNamed(name string) reference.Named {
	n, err := reference.ParseNamed(name)
	if err != nil {
		return nil
	}
	return n
}

func (c *apiClient) ImageInspect(ctx context.Context, ref string, _ ...client.ImageInspectOption) (client.ImageInspectResult, error) {
	var res client.ImageInspectResult
	err := c.eng.withSession(ctx, func(ctx context.Context, s *session) error {
		ri, err := resolveImage(ctx, s, ref, nil)
		if err != nil {
			return err
		}
		all, err := s.images().List(ctx)
		if err != nil {
			return err
		}
		var repoTags, repoDigests []string
		for _, img := range all {
			desc, err := img.Config(ctx, s.content(), ri.platform)
			if err != nil || desc.Digest != ri.configDesc.Digest {
				continue
			}
			named, err := reference.ParseNamed(img.Name)
			if err != nil {
				continue
			}
			if tagged, ok := named.(reference.Tagged); ok {
				repoTags = append(repoTags, reference.FamiliarName(named)+":"+tagged.Tag())
			}
			repoDigests = append(repoDigests, reference.FamiliarName(named)+"@"+img.Target.Digest.String())
		}
		size, _ := ri.img.Size(ctx, s.content(), ri.platform)
		var created string
		if ri.config.Created != nil {
			created = ri.config.Created.UTC().Format(time.RFC3339Nano)
		}
		layers := make([]string, 0, len(ri.config.RootFS.DiffIDs))
		for _, d := range ri.config.RootFS.DiffIDs {
			layers = append(layers, d.String())
		}
		cfg := ri.config.Config
		target := ri.img.Target
		res.InspectResponse = image.InspectResponse{
			ID:           ri.ID(),
			RepoTags:     repoTags,
			RepoDigests:  repoDigests,
			Created:      created,
			Author:       ri.config.Author,
			Config:       &cfg,
			Architecture: ri.config.Architecture,
			Variant:      ri.config.Variant,
			Os:           ri.config.OS,
			OsVersion:    ri.config.OSVersion,
			Size:         size,
			RootFS: image.RootFS{
				Type:   ri.config.RootFS.Type,
				Layers: layers,
			},
			Descriptor: &target,
		}
		return nil
	})
	return res, err
}

func (c *apiClient) ImageHistory(ctx context.Context, ref string, _ ...client.ImageHistoryOption) (client.ImageHistoryResult, error) {
	var res client.ImageHistoryResult
	err := c.eng.withSession(ctx, func(ctx context.Context, s *session) error {
		ri, err := resolveImage(ctx, s, ref, nil)
		if err != nil {
			return err
		}
		manifest, err := images.Manifest(ctx, s.content(), ri.img.Target, ri.platform)
		if err != nil {
			return err
		}
		layerIdx := 0
		for i := len(ri.config.History) - 1; i >= 0; i-- {
			h := ri.config.History[i]
			var created int64
			if h.Created != nil {
				created = h.Created.Unix()
			}
			item := image.HistoryResponseItem{
				ID:        "<missing>",
				Created:   created,
				CreatedBy: h.CreatedBy,
				Comment:   h.Comment,
			}
			if !h.EmptyLayer {
				n := len(manifest.Layers) - 1 - layerIdx
				if n >= 0 && n < len(manifest.Layers) {
					item.Size = manifest.Layers[n].Size
					item.ID = manifest.Layers[n].Digest.String()
				}
				layerIdx++
			}
			res.Items = append(res.Items, item)
		}
		if len(res.Items) > 0 {
			res.Items[0].ID = ri.ID()
		}
		return nil
	})
	return res, err
}

func (c *apiClient) ImageTag(ctx context.Context, options client.ImageTagOptions) (client.ImageTagResult, error) {
	err := c.eng.withSession(ctx, func(ctx context.Context, s *session) error {
		ri, err := resolveImage(ctx, s, options.Source, nil)
		if err != nil {
			return err
		}
		target, err := normalizeImageRef(options.Target)
		if err != nil {
			return err
		}
		target = reference.TagNameOnly(target)
		img := images.Image{Name: target.String(), Target: ri.img.Target, Labels: ri.img.Labels}
		if _, err := s.images().Create(ctx, img); err != nil {
			if !cerrdefs.IsAlreadyExists(err) {
				return err
			}
			if _, err := s.images().Update(ctx, img); err != nil {
				return err
			}
		}
		return nil
	})
	return client.ImageTagResult{}, err
}

//nolint:gocyclo // untagging and deletion have several conditions
func (c *apiClient) ImageRemove(ctx context.Context, ref string, options client.ImageRemoveOptions) (client.ImageRemoveResult, error) {
	var res client.ImageRemoveResult
	err := c.eng.withSession(ctx, func(ctx context.Context, s *session) error {
		ri, err := resolveImage(ctx, s, ref, nil)
		if err != nil {
			return err
		}
		all, err := s.images().List(ctx)
		if err != nil {
			return err
		}
		// Collect all names pointing at the same image ID.
		var sameID []images.Image
		for _, img := range all {
			desc, err := img.Config(ctx, s.content(), ri.platform)
			if err == nil && desc.Digest == ri.configDesc.Digest {
				sameID = append(sameID, img)
			}
		}
		// Refuse to remove images used by containers unless forced.
		if !options.Force {
			ctrs, err := s.containers().List(ctx)
			if err != nil {
				return err
			}
			for _, ctr := range ctrs {
				for _, img := range sameID {
					if ctr.Image == img.Name {
						return fmt.Errorf("conflict: unable to remove image %s: container %s is using it: %w", familiarizeImageRef(img.Name), shortID(ctr.ID), cerrdefs.ErrConflict)
					}
				}
			}
		}
		toDelete := sameID
		// "docker rmi name:tag" with multiple tags only untags, unless the
		// reference was an ID or force was given.
		if !isImageID(ref) && len(sameID) > 1 && !options.Force {
			toDelete = []images.Image{ri.img}
		}
		for _, img := range toDelete {
			if err := s.images().Delete(ctx, img.Name); err != nil && !cerrdefs.IsNotFound(err) {
				return err
			}
			res.Items = append(res.Items, image.DeleteResponse{Untagged: familiarizeImageRef(img.Name)})
		}
		if len(toDelete) == len(sameID) {
			res.Items = append(res.Items, image.DeleteResponse{Deleted: ri.ID()})
		}
		return s.gc(ctx)
	})
	return res, err
}

func (c *apiClient) ImagePrune(ctx context.Context, opts client.ImagePruneOptions) (client.ImagePruneResult, error) {
	var res client.ImagePruneResult
	err := c.eng.withSession(ctx, func(ctx context.Context, s *session) error {
		all, err := s.images().List(ctx)
		if err != nil {
			return err
		}
		ctrs, err := s.containers().List(ctx)
		if err != nil {
			return err
		}
		inUse := map[string]bool{}
		for _, ctr := range ctrs {
			inUse[ctr.Image] = true
		}
		dangling := true
		for v := range opts.Filters["dangling"] {
			if v == "false" || v == "0" {
				dangling = false
			}
		}
		if dangling {
			// Images are always tagged in this backend, so there is nothing to prune.
			return nil
		}
		for _, img := range all {
			if inUse[img.Name] {
				continue
			}
			size, _ := img.Size(ctx, s.content(), platforms.All)
			if err := s.images().Delete(ctx, img.Name); err != nil {
				return err
			}
			res.Report.ImagesDeleted = append(res.Report.ImagesDeleted, image.DeleteResponse{Untagged: familiarizeImageRef(img.Name)})
			res.Report.SpaceReclaimed += uint64(size)
		}
		return s.gc(ctx)
	})
	return res, err
}
