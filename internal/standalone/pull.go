//go:build linux

package standalone

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/containerd/containerd/v2/core/content"
	"github.com/containerd/containerd/v2/core/diff/apply"
	"github.com/containerd/containerd/v2/core/images"
	"github.com/containerd/containerd/v2/core/leases"
	"github.com/containerd/containerd/v2/core/remotes"
	"github.com/containerd/containerd/v2/core/remotes/docker"
	"github.com/containerd/containerd/v2/core/unpack"
	cerrdefs "github.com/containerd/errdefs"
	"github.com/containerd/log"
	"github.com/containerd/platforms"
	"github.com/distribution/reference"
	"github.com/moby/moby/api/types/jsonstream"
	"github.com/moby/moby/api/types/registry"
	"github.com/moby/moby/client"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// newResolver builds a registry resolver using the credentials passed by the
// CLI (if any). Registries on localhost are contacted over plain HTTP when
// TLS is unavailable, matching Docker's default insecure-registry behaviour.
func newResolver(auth registry.AuthConfig, tracker docker.StatusTracker) remotes.Resolver {
	creds := func(host string) (string, string, error) {
		if auth.IdentityToken != "" {
			return "", auth.IdentityToken, nil
		}
		if auth.Username == "" && auth.Password == "" {
			return "", "", nil
		}
		return auth.Username, auth.Password, nil
	}
	authorizer := docker.NewDockerAuthorizer(
		docker.WithAuthCreds(creds),
		docker.WithAuthClient(http.DefaultClient),
	)
	return docker.NewResolver(docker.ResolverOptions{
		Hosts: docker.ConfigureDefaultRegistries(
			docker.WithAuthorizer(authorizer),
			docker.WithPlainHTTP(docker.MatchLocalhost),
			docker.WithClient(http.DefaultClient),
		),
		Tracker: tracker,
	})
}

func (c *apiClient) ImagePull(ctx context.Context, ref string, options client.ImagePullOptions) (client.ImagePullResponse, error) {
	auth, err := decodeAuthConfig(options.RegistryAuth)
	if err != nil {
		return nil, err
	}
	named, err := normalizeImageRef(ref)
	if err != nil {
		return nil, err
	}
	named = reference.TagNameOnly(named)

	var matcher platforms.MatchComparer
	switch {
	case options.All:
		matcher = platforms.All
	case len(options.Platforms) > 0:
		matcher = platforms.Any(options.Platforms...)
	default:
		matcher = platforms.Default()
	}

	stream := newJSONStream()
	// The pull runs in the background; the CLI consumes progress from the
	// stream. Detach from the caller's context cancellation only to the
	// extent needed to report the final error.
	go func() {
		err := c.eng.pull(ctx, named, matcher, auth, stream)
		stream.finish(err)
	}()
	return stream, nil
}

// pull fetches an image into the content store, unpacks it into the
// snapshotter and records it in the image store.
func (e *engine) pull(ctx context.Context, named reference.Named, matcher platforms.MatchComparer, auth registry.AuthConfig, stream *jsonStream) error {
	return e.withSession(ctx, func(ctx context.Context, s *session) error {
		name := named.String()
		familiar := reference.FamiliarString(named)
		if tagged, ok := named.(reference.Tagged); ok {
			stream.status(tagged.Tag(), "Pulling from "+reference.FamiliarName(named))
		}

		lm := s.leases()
		lease, err := lm.Create(ctx, leases.WithRandomID(), leases.WithExpiration(24*time.Hour))
		if err != nil {
			return err
		}
		ctx = leases.WithLease(ctx, lease.ID)
		defer func() { _ = lm.Delete(context.WithoutCancel(ctx), lease) }()

		tracker := docker.NewInMemoryTracker()
		resolver := newResolver(auth, tracker)
		resolvedName, desc, err := resolver.Resolve(ctx, name)
		if err != nil {
			return fmt.Errorf("resolving %s: %w", familiar, err)
		}
		fetcher, err := resolver.Fetcher(ctx, resolvedName)
		if err != nil {
			return err
		}

		store := s.content()
		jobs := newPullJobs()
		trackHandler := images.HandlerFunc(func(_ context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
			jobs.add(desc)
			return nil, nil
		})

		children := images.ChildrenHandler(store)
		children = images.SetChildrenLabels(store, children)
		children = images.FilterPlatforms(children, matcher)
		if matcher != platforms.All {
			children = images.LimitManifests(children, matcher, 1)
		}
		distSource, err := docker.AppendDistributionSourceLabel(store, name)
		if err != nil {
			return err
		}
		handler := images.Handlers(trackHandler, remotes.FetchHandler(store, fetcher), children, distSource)

		unpacker, err := unpack.NewUnpacker(ctx, store, unpack.WithUnpackPlatform(unpack.Platform{
			Platform:       matcher,
			SnapshotterKey: e.cfg.snapshotter,
			Snapshotter:    s.snapshotter(),
			Applier:        apply.NewFileSystemApplier(store),
		}))
		if err != nil {
			return err
		}

		progressCtx, stopProgress := context.WithCancel(ctx)
		progressDone := make(chan struct{})
		go func() {
			defer close(progressDone)
			reportPullProgress(progressCtx, jobs, store, stream)
		}()

		dispatchErr := images.Dispatch(ctx, unpacker.Unpack(handler), nil, desc)
		_, unpackErr := unpacker.Wait()
		stopProgress()
		<-progressDone
		if dispatchErr != nil {
			return dispatchErr
		}
		if unpackErr != nil {
			return unpackErr
		}

		img := images.Image{Name: name, Target: desc}
		created, err := s.images().Create(ctx, img)
		if err != nil {
			if !cerrdefs.IsAlreadyExists(err) {
				return err
			}
			created, err = s.images().Update(ctx, img)
			if err != nil {
				return err
			}
		}
		ri, err := readImage(ctx, s, created, matcher)
		if err == nil {
			stream.status("", "Digest: "+desc.Digest.String())
			stream.status("", "Status: Downloaded newer image for "+familiar)
			log.G(ctx).WithField("image", familiar).WithField("id", ri.ID()).Debug("pulled image")
		} else {
			stream.status("", "Digest: "+desc.Digest.String())
			stream.status("", "Status: Downloaded image for "+familiar)
		}
		return nil
	})
}

// pullJobs tracks the descriptors encountered during a pull for progress
// reporting.
type pullJobs struct {
	mu    sync.Mutex
	descs []ocispec.Descriptor
	seen  map[string]struct{}
}

func newPullJobs() *pullJobs {
	return &pullJobs{seen: map[string]struct{}{}}
}

func (j *pullJobs) add(desc ocispec.Descriptor) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if _, ok := j.seen[desc.Digest.String()]; ok {
		return
	}
	j.seen[desc.Digest.String()] = struct{}{}
	j.descs = append(j.descs, desc)
}

func (j *pullJobs) list() []ocispec.Descriptor {
	j.mu.Lock()
	defer j.mu.Unlock()
	return append([]ocispec.Descriptor(nil), j.descs...)
}

// reportPullProgress periodically inspects the content store's active
// ingests and emits Docker-style progress messages for each layer.
//
//nolint:gocyclo // maps content-store states onto Docker's progress messages
func reportPullProgress(ctx context.Context, jobs *pullJobs, store content.Store, stream *jsonStream) {
	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()
	start := time.Now()
	last := map[string]string{}
	emit := func(id, status string, cur, total int64) {
		key := fmt.Sprintf("%s|%d|%d", status, cur, total)
		if last[id] == key {
			return
		}
		last[id] = key
		if total > 0 && status == "Downloading" {
			stream.progress(id, status, cur, total)
			return
		}
		stream.status(id, status)
	}
	done := false
	for {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			done = true
		}
		active := map[string]content.Status{}
		if !done {
			statuses, err := store.ListStatuses(ctx, "")
			if err == nil {
				for _, st := range statuses {
					active[st.Ref] = st
				}
			}
		}
		for _, desc := range jobs.list() {
			if !images.IsLayerType(desc.MediaType) {
				continue
			}
			id := shortID(desc.Digest.Encoded())
			key := remotes.MakeRefKey(ctx, desc)
			if st, ok := active[key]; ok {
				emit(id, "Downloading", st.Offset, st.Total)
				continue
			}
			info, err := store.Info(ctx, desc.Digest)
			switch {
			case err != nil:
				if done {
					emit(id, "Pull complete", 0, 0)
				} else {
					emit(id, "Waiting", 0, 0)
				}
			case info.CreatedAt.After(start):
				emit(id, "Pull complete", 0, 0)
			default:
				emit(id, "Already exists", 0, 0)
			}
		}
		if done {
			return
		}
	}
}

func (c *apiClient) ImagePush(ctx context.Context, ref string, options client.ImagePushOptions) (client.ImagePushResponse, error) {
	auth, err := decodeAuthConfig(options.RegistryAuth)
	if err != nil {
		return nil, err
	}
	named, err := normalizeImageRef(ref)
	if err != nil {
		return nil, err
	}
	named = reference.TagNameOnly(named)
	var matcher platforms.MatchComparer
	if options.Platform != nil {
		matcher = platforms.Only(*options.Platform)
	}

	stream := newJSONStream()
	go func() {
		err := c.eng.withSession(ctx, func(ctx context.Context, s *session) error {
			img, err := s.images().Get(ctx, named.String())
			if err != nil {
				return err
			}
			if tagged, ok := named.(reference.Tagged); ok {
				stream.status("", "The push refers to repository ["+reference.FamiliarName(named)+"]")
				defer func() {
					if err == nil {
						stream.status("", fmt.Sprintf("%s: digest: %s size: %d", tagged.Tag(), img.Target.Digest, img.Target.Size))
					}
				}()
			}
			tracker := docker.NewInMemoryTracker()
			resolver := newResolver(auth, tracker)
			pusher, err := resolver.Pusher(ctx, named.String())
			if err != nil {
				return err
			}
			wrapper := func(h images.Handler) images.Handler {
				return images.Handlers(images.HandlerFunc(func(_ context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
					if images.IsLayerType(desc.MediaType) {
						stream.send(jsonstream.Message{ID: shortID(desc.Digest.Encoded()), Status: "Pushing"})
					}
					return nil, nil
				}), h)
			}
			if err := remotes.PushContent(ctx, pusher, img.Target, s.content(), nil, matcher, wrapper); err != nil {
				return err
			}
			return nil
		})
		stream.finish(err)
	}()
	return stream, nil
}
