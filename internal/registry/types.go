package registry

import (
	"github.com/distribution/reference"
	"github.com/docker/docker/api/types/registry"
)

// RepositoryInfo describes a repository
type RepositoryInfo struct {
	Name reference.Named
	// Index points to registry information
	Index *registry.IndexInfo
}
