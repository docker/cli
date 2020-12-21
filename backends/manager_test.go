package backends

import (
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
)

func TestListBackend(t *testing.T) {
	// Populate a selection of directories with various shadowed and bogus/obscure plugin candidates.
	// For the purposes of this test no contents is required and permissions are irrelevant.
	dir := fs.NewDir(t, t.Name(),
		fs.WithFile("docker-aci-ecs-local-backend", ""), // backend for aci, ecs, local types
		fs.WithFile("docker-type1-backend", ""),         // backend for type1
		fs.WithFile("docker-type2-backend.exe", ""),     // backend for type2
		fs.WithFile("not-a-backend", ""),
		fs.WithFile("docker-plugin", ""),
		fs.WithDir("ignored1"),
	)
	defer dir.Remove()

	candidates, err := listBackendsFrom(dir.Path(), fakeMetadata)
	assert.NilError(t, err)
	exp := []Backend{
		{Name: "aci-ecs-local-backend", Path: filepath.Join(dir.Path(), "docker-aci-ecs-local-backend"), Version: "1.0", SupportedTypes: []string{"aci", "ecs", "local"}},
		{Name: "type1-backend", Path: filepath.Join(dir.Path(), "docker-type1-backend"), Version: "1.0", SupportedTypes: []string{"type1"}},
		{Name: "type2-backend", Path: filepath.Join(dir.Path(), "docker-type2-backend.exe"), Version: "1.0", SupportedTypes: []string{"type2"}},
	}

	assert.DeepEqual(t, candidates, exp)
}
