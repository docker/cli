package registry

import (
	"testing"

	"github.com/distribution/reference"
	"github.com/moby/moby/api/types/registry"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestNewIndexInfo(t *testing.T) {
	type staticRepositoryInfo struct {
		Index         *registry.IndexInfo
		RemoteName    string
		CanonicalName string
		LocalName     string
	}

	tests := map[string]staticRepositoryInfo{
		"fooo/bar": {
			Index: &registry.IndexInfo{
				Name:     IndexName,
				Official: true,
				Secure:   true,
			},
			RemoteName:    "fooo/bar",
			LocalName:     "fooo/bar",
			CanonicalName: "docker.io/fooo/bar",
		},
		"library/ubuntu": {
			Index: &registry.IndexInfo{
				Name:     IndexName,
				Official: true,
				Secure:   true,
			},
			RemoteName:    "library/ubuntu",
			LocalName:     "ubuntu",
			CanonicalName: "docker.io/library/ubuntu",
		},
		"nonlibrary/ubuntu": {
			Index: &registry.IndexInfo{
				Name:     IndexName,
				Official: true,
				Secure:   true,
			},
			RemoteName:    "nonlibrary/ubuntu",
			LocalName:     "nonlibrary/ubuntu",
			CanonicalName: "docker.io/nonlibrary/ubuntu",
		},
		"ubuntu": {
			Index: &registry.IndexInfo{
				Name:     IndexName,
				Official: true,
				Secure:   true,
			},
			RemoteName:    "library/ubuntu",
			LocalName:     "ubuntu",
			CanonicalName: "docker.io/library/ubuntu",
		},
		"other/library": {
			Index: &registry.IndexInfo{
				Name:     IndexName,
				Official: true,
				Secure:   true,
			},
			RemoteName:    "other/library",
			LocalName:     "other/library",
			CanonicalName: "docker.io/other/library",
		},
		"127.0.0.1:8000/private/moonbase": {
			Index: &registry.IndexInfo{
				Name:     "127.0.0.1:8000",
				Official: false,
				Secure:   false,
			},
			RemoteName:    "private/moonbase",
			LocalName:     "127.0.0.1:8000/private/moonbase",
			CanonicalName: "127.0.0.1:8000/private/moonbase",
		},
		"127.0.0.1:8000/privatebase": {
			Index: &registry.IndexInfo{
				Name:     "127.0.0.1:8000",
				Official: false,
				Secure:   false,
			},
			RemoteName:    "privatebase",
			LocalName:     "127.0.0.1:8000/privatebase",
			CanonicalName: "127.0.0.1:8000/privatebase",
		},
		"[::1]:8000/private/moonbase": {
			Index: &registry.IndexInfo{
				Name:     "[::1]:8000",
				Official: false,
				Secure:   false,
			},
			RemoteName:    "private/moonbase",
			LocalName:     "[::1]:8000/private/moonbase",
			CanonicalName: "[::1]:8000/private/moonbase",
		},
		"[::1]:8000/privatebase": {
			Index: &registry.IndexInfo{
				Name:     "[::1]:8000",
				Official: false,
				Secure:   false,
			},
			RemoteName:    "privatebase",
			LocalName:     "[::1]:8000/privatebase",
			CanonicalName: "[::1]:8000/privatebase",
		},
		// IPv6 only has a single loopback address, so ::2 is not a loopback,
		// hence not marked "insecure".
		"[::2]:8000/private/moonbase": {
			Index: &registry.IndexInfo{
				Name:     "[::2]:8000",
				Official: false,
				Secure:   true,
			},
			RemoteName:    "private/moonbase",
			LocalName:     "[::2]:8000/private/moonbase",
			CanonicalName: "[::2]:8000/private/moonbase",
		},
		// IPv6 only has a single loopback address, so ::2 is not a loopback,
		// hence not marked "insecure".
		"[::2]:8000/privatebase": {
			Index: &registry.IndexInfo{
				Name:     "[::2]:8000",
				Official: false,
				Secure:   true,
			},
			RemoteName:    "privatebase",
			LocalName:     "[::2]:8000/privatebase",
			CanonicalName: "[::2]:8000/privatebase",
		},
		"localhost:8000/private/moonbase": {
			Index: &registry.IndexInfo{
				Name:     "localhost:8000",
				Official: false,
				Secure:   false,
			},
			RemoteName:    "private/moonbase",
			LocalName:     "localhost:8000/private/moonbase",
			CanonicalName: "localhost:8000/private/moonbase",
		},
		"localhost:8000/privatebase": {
			Index: &registry.IndexInfo{
				Name:     "localhost:8000",
				Official: false,
				Secure:   false,
			},
			RemoteName:    "privatebase",
			LocalName:     "localhost:8000/privatebase",
			CanonicalName: "localhost:8000/privatebase",
		},
		"example.com/private/moonbase": {
			Index: &registry.IndexInfo{
				Name:     "example.com",
				Official: false,
				Secure:   true,
			},
			RemoteName:    "private/moonbase",
			LocalName:     "example.com/private/moonbase",
			CanonicalName: "example.com/private/moonbase",
		},
		"example.com/privatebase": {
			Index: &registry.IndexInfo{
				Name:     "example.com",
				Official: false,
				Secure:   true,
			},
			RemoteName:    "privatebase",
			LocalName:     "example.com/privatebase",
			CanonicalName: "example.com/privatebase",
		},
		"example.com:8000/private/moonbase": {
			Index: &registry.IndexInfo{
				Name:     "example.com:8000",
				Official: false,
				Secure:   true,
			},
			RemoteName:    "private/moonbase",
			LocalName:     "example.com:8000/private/moonbase",
			CanonicalName: "example.com:8000/private/moonbase",
		},
		"example.com:8000/privatebase": {
			Index: &registry.IndexInfo{
				Name:     "example.com:8000",
				Official: false,
				Secure:   true,
			},
			RemoteName:    "privatebase",
			LocalName:     "example.com:8000/privatebase",
			CanonicalName: "example.com:8000/privatebase",
		},
		"localhost/private/moonbase": {
			Index: &registry.IndexInfo{
				Name:     "localhost",
				Official: false,
				Secure:   false,
			},
			RemoteName:    "private/moonbase",
			LocalName:     "localhost/private/moonbase",
			CanonicalName: "localhost/private/moonbase",
		},
		"localhost/privatebase": {
			Index: &registry.IndexInfo{
				Name:     "localhost",
				Official: false,
				Secure:   false,
			},
			RemoteName:    "privatebase",
			LocalName:     "localhost/privatebase",
			CanonicalName: "localhost/privatebase",
		},
		IndexName + "/public/moonbase": {
			Index: &registry.IndexInfo{
				Name:     IndexName,
				Official: true,
				Secure:   true,
			},
			RemoteName:    "public/moonbase",
			LocalName:     "public/moonbase",
			CanonicalName: "docker.io/public/moonbase",
		},
		"index." + IndexName + "/public/moonbase": {
			Index: &registry.IndexInfo{
				Name:     IndexName,
				Official: true,
				Secure:   true,
			},
			RemoteName:    "public/moonbase",
			LocalName:     "public/moonbase",
			CanonicalName: "docker.io/public/moonbase",
		},
		"ubuntu-12.04-base": {
			Index: &registry.IndexInfo{
				Name:     IndexName,
				Official: true,
				Secure:   true,
			},
			RemoteName:    "library/ubuntu-12.04-base",
			LocalName:     "ubuntu-12.04-base",
			CanonicalName: "docker.io/library/ubuntu-12.04-base",
		},
		IndexName + "/ubuntu-12.04-base": {
			Index: &registry.IndexInfo{
				Name:     IndexName,
				Official: true,
				Secure:   true,
			},
			RemoteName:    "library/ubuntu-12.04-base",
			LocalName:     "ubuntu-12.04-base",
			CanonicalName: "docker.io/library/ubuntu-12.04-base",
		},
		"index." + IndexName + "/ubuntu-12.04-base": {
			Index: &registry.IndexInfo{
				Name:     IndexName,
				Official: true,
				Secure:   true,
			},
			RemoteName:    "library/ubuntu-12.04-base",
			LocalName:     "ubuntu-12.04-base",
			CanonicalName: "docker.io/library/ubuntu-12.04-base",
		},
	}

	for reposName, expected := range tests {
		t.Run(reposName, func(t *testing.T) {
			named, err := reference.ParseNormalizedNamed(reposName)
			assert.NilError(t, err)

			indexInfo := NewIndexInfo(named)
			repoInfoName := reference.TrimNamed(named)

			assert.Check(t, is.DeepEqual(indexInfo, expected.Index))
			assert.Check(t, is.Equal(reference.Path(repoInfoName), expected.RemoteName))
			assert.Check(t, is.Equal(reference.FamiliarName(repoInfoName), expected.LocalName))
			assert.Check(t, is.Equal(repoInfoName.Name(), expected.CanonicalName))
		})
	}
}
