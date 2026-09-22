//go:build linux

package standalone

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/distribution/reference"
	"github.com/moby/moby/api/types/registry"
	"github.com/opencontainers/go-digest"
)

// normalizeImageRef converts a Docker-style image reference (for example
// "ubuntu" or "ubuntu:24.04") to the fully-qualified form used as image name
// in the containerd image store ("docker.io/library/ubuntu:24.04").
func normalizeImageRef(ref string) (reference.Named, error) {
	named, err := reference.ParseDockerRef(ref)
	if err != nil {
		return nil, fmt.Errorf("invalid reference %q: %w: %w", ref, err, cerrdefs.ErrInvalidArgument)
	}
	return named, nil
}

// familiarizeImageRef converts a fully-qualified image name back to the short
// form shown to users by the Docker CLI.
func familiarizeImageRef(name string) string {
	named, err := reference.ParseNamed(name)
	if err != nil {
		return name
	}
	return reference.FamiliarString(named)
}

// isImageID reports whether ref looks like a (possibly truncated) image ID
// rather than a reference.
func isImageID(ref string) bool {
	ref = strings.TrimPrefix(ref, "sha256:")
	if len(ref) < 4 || len(ref) > 64 {
		return false
	}
	for _, c := range ref {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// matchesImageID reports whether the given digest matches the (possibly
// truncated) user-supplied ID.
func matchesImageID(dgst digest.Digest, id string) bool {
	id = strings.TrimPrefix(id, "sha256:")
	return strings.HasPrefix(dgst.Encoded(), id)
}

// decodeAuthConfig decodes the base64url-encoded registry credentials passed
// by the CLI in the X-Registry-Auth header format.
func decodeAuthConfig(encoded string) (registry.AuthConfig, error) {
	var ac registry.AuthConfig
	if encoded == "" {
		return ac, nil
	}
	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		data, err = base64.RawURLEncoding.DecodeString(encoded)
		if err != nil {
			return ac, fmt.Errorf("invalid registry auth: %w", err)
		}
	}
	if err := json.Unmarshal(data, &ac); err != nil {
		return ac, fmt.Errorf("invalid registry auth: %w", err)
	}
	return ac, nil
}
