package backends

import (
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
)

func TestListBackend(t *testing.T) {
	// Populate a selection of directories with various shadowed and bogus/obscure plugin candidates.
	// For the purposes of this test no contents is required and permissions are irrelevant.
	dir := fs.NewDir(t, t.Name(),
		fs.WithFile("docker-aci-local-backend", ""),      // backend for aci, local types
		fs.WithFile("docker-ecs-backend.exe", ""),        // backend for ecs
		fs.WithFile("docker-notallowed-backend.exe", ""), // type not allowed
		fs.WithFile("docker-local-backend.exe", ""),      // binary not allowed
		fs.WithFile("not-a-backend", ""),
		fs.WithFile("docker-plugin", ""),
		fs.WithDir("ignored1"),
	)
	defer dir.Remove()

	candidates, err := listBackendsFrom(dir.Path(), fakeMetadata, map[string][]string{"docker-aci-local-backend": {"aci", "local"}, "docker-ecs-backend": {"ecs"}})
	exp := []Backend{
		{Name: "aci-local-backend", Path: filepath.Join(dir.Path(), "docker-aci-local-backend"), Version: "1.0", SupportedTypes: []string{"aci", "local"}},
		{Name: "ecs-backend", Path: filepath.Join(dir.Path(), "docker-ecs-backend.exe"), Version: "1.0", SupportedTypes: []string{"ecs"}},
	}

	assert.NilError(t, err)
	assert.DeepEqual(t, candidates, exp)
}

func fakeMetadata(binary string) (BackendMetadata, error) {
	name := strings.TrimPrefix(filepath.Base(binary), backendPrefix)
	withoutExe := strings.TrimSuffix(name, ".exe")
	return BackendMetadata{Name: withoutExe, Version: "1.0"}, nil
}
