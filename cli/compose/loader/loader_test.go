// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.22

package loader

import (
	"fmt"
	"os"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func buildConfigDetailsFromYAML(yaml string, env map[string]string) types.ConfigDetails {
	return buildConfigDetailsMultipleFiles(env, yaml)
}

func buildConfigDetailsMultipleFiles(env map[string]string, yamls ...string) types.ConfigDetails {
	workingDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	if env == nil {
		env = map[string]string{}
	}

	return types.ConfigDetails{
		WorkingDir:  workingDir,
		ConfigFiles: buildConfigFiles(yamls),
		Environment: env,
	}
}

func buildConfigFiles(yamls []string) []types.ConfigFile {
	configFiles := []types.ConfigFile{}
	for i, yaml := range yamls {
		configFiles = append(configFiles, types.ConfigFile{
			Filename: fmt.Sprintf("filename%d.yml", i),
			Content:  []byte(yaml),
		})
	}
	return configFiles
}

func loadYAML(yaml string) (*types.Project, error) {
	return loadYAMLWithEnv(yaml, nil)
}

func loadYAMLWithEnv(yaml string, env map[string]string) (*types.Project, error) {
	return Load(buildConfigDetailsFromYAML(yaml, env))
}

func strPtr(val string) *string {
	return &val
}

func TestUnsupportedProperties(t *testing.T) {
	config := `
version: "3"
name: unsupported-properties
services:
  web:
    image: web
    build:
     context: ./web
    links:
      - db
    pid: host
  db:
    image: db
    build:
     context: ./db
`

	configDetails := buildConfigDetailsFromYAML(config, nil)

	project, err := Load(configDetails)
	assert.NilError(t, err)

	unsupported := GetUnsupportedProperties(project.Services)
	assert.Check(t, is.DeepEqual([]string{"build", "links", "pid"}, unsupported))
}

func TestDeprecatedProperties(t *testing.T) {
	config := `
version: "3"
name: deprecated-properties
services:
  web:
    image: web
    container_name: web
  db:
    image: db
    container_name: db
    expose: ["5434"]
`

	configDetails := buildConfigDetailsFromYAML(config, nil)

	project, err := Load(configDetails)
	assert.NilError(t, err)

	deprecated := GetDeprecatedProperties(project.Services)
	assert.Check(t, is.Len(deprecated, 2))
	assert.Check(t, is.Contains(deprecated, "container_name"))
	assert.Check(t, is.Contains(deprecated, "expose"))
}

func TestForbiddenProperties(t *testing.T) {
	_, err := loadYAML(`
version: "3"
name: forbidden-properties
services:
  foo:
    image: busybox
    volumes:
      - /data
    volumes_from: 
      - bar
  bar:
    extends:
      service: quick
  quick:
    image: busybox
    cpu_quota: 50000
`)

	assert.ErrorType(t, err, &ForbiddenPropertiesError{})

	props := err.(*ForbiddenPropertiesError).Properties
	assert.Check(t, is.Len(props, 2))
	assert.Check(t, is.Contains(props, "volumes_from"))
	assert.Check(t, is.Contains(props, "cpu_quota"))
}
