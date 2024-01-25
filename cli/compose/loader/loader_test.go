// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.19

package loader

import (
	"bytes"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/docker/cli/cli/compose/types"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sirupsen/logrus"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func buildConfigDetails(source map[string]interface{}, env map[string]string) types.ConfigDetails {
	workingDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	return types.ConfigDetails{
		WorkingDir: workingDir,
		ConfigFiles: []types.ConfigFile{
			{Filename: "filename.yml", Config: source},
		},
		Environment: env,
	}
}

func loadYAML(yaml string) (*types.Config, error) {
	return loadYAMLWithEnv(yaml, nil)
}

func loadYAMLWithEnv(yaml string, env map[string]string) (*types.Config, error) {
	dict, err := ParseYAML([]byte(yaml))
	if err != nil {
		return nil, err
	}

	return Load(buildConfigDetails(dict, env))
}

var sampleYAML = `
version: "3"
services:
  foo:
    image: busybox
    networks:
      with_me:
  bar:
    image: busybox
    environment:
      - FOO=1
    networks:
      - with_ipam
volumes:
  hello:
    driver: default
    driver_opts:
      beep: boop
networks:
  default:
    driver: bridge
    driver_opts:
      beep: boop
  with_ipam:
    ipam:
      driver: default
      config:
        - subnet: 172.28.0.0/16
`

var sampleDict = map[string]interface{}{
	"version": "3",
	"services": map[string]interface{}{
		"foo": map[string]interface{}{
			"image":    "busybox",
			"networks": map[string]interface{}{"with_me": nil},
		},
		"bar": map[string]interface{}{
			"image":       "busybox",
			"environment": []interface{}{"FOO=1"},
			"networks":    []interface{}{"with_ipam"},
		},
	},
	"volumes": map[string]interface{}{
		"hello": map[string]interface{}{
			"driver": "default",
			"driver_opts": map[string]interface{}{
				"beep": "boop",
			},
		},
	},
	"networks": map[string]interface{}{
		"default": map[string]interface{}{
			"driver": "bridge",
			"driver_opts": map[string]interface{}{
				"beep": "boop",
			},
		},
		"with_ipam": map[string]interface{}{
			"ipam": map[string]interface{}{
				"driver": "default",
				"config": []interface{}{
					map[string]interface{}{
						"subnet": "172.28.0.0/16",
					},
				},
			},
		},
	},
}

var samplePortsConfig = []types.ServicePortConfig{
	{
		Mode:      "ingress",
		Target:    8080,
		Published: 80,
		Protocol:  "tcp",
	},
	{
		Mode:      "ingress",
		Target:    8081,
		Published: 81,
		Protocol:  "tcp",
	},
	{
		Mode:      "ingress",
		Target:    8082,
		Published: 82,
		Protocol:  "tcp",
	},
	{
		Mode:      "ingress",
		Target:    8090,
		Published: 90,
		Protocol:  "udp",
	},
	{
		Mode:      "ingress",
		Target:    8091,
		Published: 91,
		Protocol:  "udp",
	},
	{
		Mode:      "ingress",
		Target:    8092,
		Published: 92,
		Protocol:  "udp",
	},
	{
		Mode:      "ingress",
		Target:    8500,
		Published: 85,
		Protocol:  "tcp",
	},
	{
		Mode:      "ingress",
		Target:    8600,
		Published: 0,
		Protocol:  "tcp",
	},
	{
		Target:    53,
		Published: 10053,
		Protocol:  "udp",
	},
	{
		Mode:      "host",
		Target:    22,
		Published: 10022,
	},
}

func strPtr(val string) *string {
	return &val
}

var sampleConfig = types.Config{
	Version: "3.11",
	Services: []types.ServiceConfig{
		{
			Name:        "foo",
			Image:       "busybox",
			Environment: map[string]*string{},
			Networks: map[string]*types.ServiceNetworkConfig{
				"with_me": nil,
			},
		},
		{
			Name:        "bar",
			Image:       "busybox",
			Environment: map[string]*string{"FOO": strPtr("1")},
			Networks: map[string]*types.ServiceNetworkConfig{
				"with_ipam": nil,
			},
		},
	},
	Networks: map[string]types.NetworkConfig{
		"default": {
			Driver: "bridge",
			DriverOpts: map[string]string{
				"beep": "boop",
			},
		},
		"with_ipam": {
			Ipam: types.IPAMConfig{
				Driver: "default",
				Config: []*types.IPAMPool{
					{
						Subnet: "172.28.0.0/16",
					},
				},
			},
		},
	},
	Volumes: map[string]types.VolumeConfig{
		"hello": {
			Driver: "default",
			DriverOpts: map[string]string{
				"beep": "boop",
			},
		},
	},
}

func TestParseYAML(t *testing.T) {
	dict, err := ParseYAML([]byte(sampleYAML))
	assert.NilError(t, err)
	assert.Check(t, is.DeepEqual(sampleDict, dict))
}

func TestLoad(t *testing.T) {
	actual, err := Load(buildConfigDetails(sampleDict, nil))
	assert.NilError(t, err)
	assert.Check(t, is.Equal(sampleConfig.Version, actual.Version))
	assert.Check(t, is.DeepEqual(serviceSort(sampleConfig.Services), serviceSort(actual.Services)))
	assert.Check(t, is.DeepEqual(sampleConfig.Networks, actual.Networks))
	assert.Check(t, is.DeepEqual(sampleConfig.Volumes, actual.Volumes))
}

func TestLoadExtras(t *testing.T) {
	actual, err := loadYAML(`
version: "3.7"
services:
  foo:
    image: busybox
    x-foo: bar`)
	assert.NilError(t, err)
	assert.Check(t, is.Len(actual.Services, 1))
	service := actual.Services[0]
	assert.Check(t, is.Equal("busybox", service.Image))
	extras := map[string]interface{}{
		"x-foo": "bar",
	}
	assert.Check(t, is.DeepEqual(extras, service.Extras))
}

func TestLoadV31(t *testing.T) {
	actual, err := loadYAML(`
version: "3.1"
services:
  foo:
    image: busybox
    secrets: [super]
secrets:
  super:
    external: true
`)
	assert.NilError(t, err)
	assert.Check(t, is.Len(actual.Services, 1))
	assert.Check(t, is.Len(actual.Secrets, 1))
}

func TestLoadV33(t *testing.T) {
	actual, err := loadYAML(`
version: "3.3"
services:
  foo:
    image: busybox
    credential_spec:
      file: "/foo"
    configs: [super]
configs:
  super:
    external: true
`)
	assert.NilError(t, err)
	assert.Assert(t, is.Len(actual.Services, 1))
	assert.Check(t, is.Equal(actual.Services[0].CredentialSpec.File, "/foo"))
	assert.Assert(t, is.Len(actual.Configs, 1))
}

func TestLoadV38(t *testing.T) {
	actual, err := loadYAML(`
version: "3.8"
services:
  foo:
    image: busybox
    credential_spec:
      config: "0bt9dmxjvjiqermk6xrop3ekq"
`)
	assert.NilError(t, err)
	assert.Assert(t, is.Len(actual.Services, 1))
	assert.Check(t, is.Equal(actual.Services[0].CredentialSpec.Config, "0bt9dmxjvjiqermk6xrop3ekq"))
}

func TestParseAndLoad(t *testing.T) {
	actual, err := loadYAML(sampleYAML)
	assert.NilError(t, err)
	assert.Check(t, is.DeepEqual(serviceSort(sampleConfig.Services), serviceSort(actual.Services)))
	assert.Check(t, is.DeepEqual(sampleConfig.Networks, actual.Networks))
	assert.Check(t, is.DeepEqual(sampleConfig.Volumes, actual.Volumes))
}

func TestInvalidTopLevelObjectType(t *testing.T) {
	_, err := loadYAML("1")
	assert.Check(t, is.ErrorContains(err, "top-level object must be a mapping"))

	_, err = loadYAML(`"hello"`)
	assert.Check(t, is.ErrorContains(err, "top-level object must be a mapping"))

	_, err = loadYAML(`["hello"]`)
	assert.Check(t, is.ErrorContains(err, "top-level object must be a mapping"))
}

func TestNonStringKeys(t *testing.T) {
	_, err := loadYAML(`
version: "3"
123:
  foo:
    image: busybox
`)
	assert.Check(t, is.ErrorContains(err, "non-string key at top level: 123"))

	_, err = loadYAML(`
version: "3"
services:
  foo:
    image: busybox
  123:
    image: busybox
`)
	assert.Check(t, is.ErrorContains(err, "non-string key in services: 123"))

	_, err = loadYAML(`
version: "3"
services:
  foo:
    image: busybox
networks:
  default:
    ipam:
      config:
        - 123: oh dear
`)
	assert.Check(t, is.ErrorContains(err, "non-string key in networks.default.ipam.config[0]: 123"))

	_, err = loadYAML(`
version: "3"
services:
  dict-env:
    image: busybox
    environment:
      1: FOO
`)
	assert.Check(t, is.ErrorContains(err, "non-string key in services.dict-env.environment: 1"))
}

func TestSupportedVersion(t *testing.T) {
	_, err := loadYAML(`
version: "3"
services:
  foo:
    image: busybox
`)
	assert.Check(t, err)

	_, err = loadYAML(`
version: "3.0"
services:
  foo:
    image: busybox
`)
	assert.Check(t, err)
}

func TestUnsupportedVersion(t *testing.T) {
	_, err := loadYAML(`
version: "2"
services:
  foo:
    image: busybox
`)
	assert.Check(t, is.ErrorContains(err, "version"))

	_, err = loadYAML(`
version: "2.0"
services:
  foo:
    image: busybox
`)
	assert.Check(t, is.ErrorContains(err, "version"))
}

func TestInvalidVersion(t *testing.T) {
	_, err := loadYAML(`
version: 3
services:
  foo:
    image: busybox
`)
	assert.Check(t, is.ErrorContains(err, "version must be a string"))
}

func TestV1Unsupported(t *testing.T) {
	_, err := loadYAML(`
foo:
  image: busybox
`)
	assert.Check(t, is.ErrorContains(err, "(root) Additional property foo is not allowed"))

	_, err = loadYAML(`
version: "1.0"
foo:
  image: busybox
`)

	assert.Check(t, is.ErrorContains(err, "unsupported Compose file version: 1.0"))
}

func TestNonMappingObject(t *testing.T) {
	_, err := loadYAML(`
version: "3"
services:
  - foo:
      image: busybox
`)
	assert.Check(t, is.ErrorContains(err, "services must be a mapping"))

	_, err = loadYAML(`
version: "3"
services:
  foo: busybox
`)
	assert.Check(t, is.ErrorContains(err, "services.foo must be a mapping"))

	_, err = loadYAML(`
version: "3"
networks:
  - default:
      driver: bridge
`)
	assert.Check(t, is.ErrorContains(err, "networks must be a mapping"))

	_, err = loadYAML(`
version: "3"
networks:
  default: bridge
`)
	assert.Check(t, is.ErrorContains(err, "networks.default must be a mapping"))

	_, err = loadYAML(`
version: "3"
volumes:
  - data:
      driver: local
`)
	assert.Check(t, is.ErrorContains(err, "volumes must be a mapping"))

	_, err = loadYAML(`
version: "3"
volumes:
  data: local
`)
	assert.Check(t, is.ErrorContains(err, "volumes.data must be a mapping"))
}

func TestNonStringImage(t *testing.T) {
	_, err := loadYAML(`
version: "3"
services:
  foo:
    image: ["busybox", "latest"]
`)
	assert.Check(t, is.ErrorContains(err, "services.foo.image must be a string"))
}

func TestIgnoreBuildProperties(t *testing.T) {
	_, err := loadYAML(`
services:
  foo:
    image: busybox
    build:
      unsupported_prop: foo
`)
	assert.NilError(t, err)
}

func TestLoadWithEnvironment(t *testing.T) {
	config, err := loadYAMLWithEnv(`
version: "3"
services:
  dict-env:
    image: busybox
    environment:
      FOO: "1"
      BAR: 2
      BAZ: 2.5
      QUX:
      QUUX:
  list-env:
    image: busybox
    environment:
      - FOO=1
      - BAR=2
      - BAZ=2.5
      - QUX=
      - QUUX
`, map[string]string{"QUX": "qux"})
	assert.NilError(t, err)

	expected := types.MappingWithEquals{
		"FOO":  strPtr("1"),
		"BAR":  strPtr("2"),
		"BAZ":  strPtr("2.5"),
		"QUX":  strPtr("qux"),
		"QUUX": nil,
	}

	assert.Check(t, is.Equal(2, len(config.Services)))

	for _, service := range config.Services {
		assert.Check(t, is.DeepEqual(expected, service.Environment))
	}
}

func TestInvalidEnvironmentValue(t *testing.T) {
	_, err := loadYAML(`
version: "3"
services:
  dict-env:
    image: busybox
    environment:
      FOO: ["1"]
`)
	assert.Check(t, is.ErrorContains(err, "services.dict-env.environment.FOO must be a string, number or null"))
}

func TestInvalidEnvironmentObject(t *testing.T) {
	_, err := loadYAML(`
version: "3"
services:
  dict-env:
    image: busybox
    environment: "FOO=1"
`)
	assert.Check(t, is.ErrorContains(err, "services.dict-env.environment must be a mapping"))
}

func TestLoadWithEnvironmentInterpolation(t *testing.T) {
	home := "/home/foo"
	config, err := loadYAMLWithEnv(`
version: "3"
services:
  test:
    image: busybox
    labels:
      - home1=$HOME
      - home2=${HOME}
      - nonexistent=$NONEXISTENT
      - default=${NONEXISTENT-default}
networks:
  test:
    driver: $HOME
volumes:
  test:
    driver: $HOME
`, map[string]string{
		"HOME": home,
		"FOO":  "foo",
	})

	assert.NilError(t, err)

	expectedLabels := types.Labels{
		"home1":       home,
		"home2":       home,
		"nonexistent": "",
		"default":     "default",
	}

	assert.Check(t, is.DeepEqual(expectedLabels, config.Services[0].Labels))
	assert.Check(t, is.Equal(home, config.Networks["test"].Driver))
	assert.Check(t, is.Equal(home, config.Volumes["test"].Driver))
}

func TestLoadWithInterpolationCastFull(t *testing.T) {
	dict, err := ParseYAML([]byte(`
version: "3.8"
services:
  web:
    configs:
      - source: appconfig
        mode: $theint
    secrets:
      - source: super
        mode: $theint
    healthcheck:
      retries: ${theint}
      disable: $thebool
    deploy:
      replicas: $theint
      update_config:
        parallelism: $theint
        max_failure_ratio: $thefloat
      rollback_config:
        parallelism: $theint
        max_failure_ratio: $thefloat
      restart_policy:
        max_attempts: $theint
      placement:
        max_replicas_per_node: $theint
    ports:
      - $theint
      - "34567"
      - target: $theint
        published: $theint
    ulimits:
      nproc: $theint
      nofile:
        hard: $theint
        soft: $theint
    privileged: $thebool
    read_only: $thebool
    stdin_open: ${thebool}
    tty: $thebool
    volumes:
      - source: data
        type: volume
        read_only: $thebool
        volume:
          nocopy: $thebool

configs:
  appconfig:
    external: $thebool
secrets:
  super:
    external: $thebool
volumes:
  data:
    external: $thebool
networks:
  front:
    external: $thebool
    internal: $thebool
    attachable: $thebool

`))
	assert.NilError(t, err)
	env := map[string]string{
		"theint":   "555",
		"thefloat": "3.14",
		"thebool":  "true",
	}

	config, err := Load(buildConfigDetails(dict, env))
	assert.NilError(t, err)
	expected := &types.Config{
		Filename: "filename.yml",
		Version:  "3.8",
		Services: []types.ServiceConfig{
			{
				Name: "web",
				Configs: []types.ServiceConfigObjConfig{
					{
						Source: "appconfig",
						Mode:   uint32Ptr(555),
					},
				},
				Secrets: []types.ServiceSecretConfig{
					{
						Source: "super",
						Mode:   uint32Ptr(555),
					},
				},
				HealthCheck: &types.HealthCheckConfig{
					Retries: uint64Ptr(555),
					Disable: true,
				},
				Deploy: types.DeployConfig{
					Replicas: uint64Ptr(555),
					UpdateConfig: &types.UpdateConfig{
						Parallelism:     uint64Ptr(555),
						MaxFailureRatio: 3.14,
					},
					RollbackConfig: &types.UpdateConfig{
						Parallelism:     uint64Ptr(555),
						MaxFailureRatio: 3.14,
					},
					RestartPolicy: &types.RestartPolicy{
						MaxAttempts: uint64Ptr(555),
					},
					Placement: types.Placement{
						MaxReplicas: 555,
					},
				},
				Ports: []types.ServicePortConfig{
					{Target: 555, Mode: "ingress", Protocol: "tcp"},
					{Target: 34567, Mode: "ingress", Protocol: "tcp"},
					{Target: 555, Published: 555},
				},
				Ulimits: map[string]*types.UlimitsConfig{
					"nproc":  {Single: 555},
					"nofile": {Hard: 555, Soft: 555},
				},
				Privileged: true,
				ReadOnly:   true,
				StdinOpen:  true,
				Tty:        true,
				Volumes: []types.ServiceVolumeConfig{
					{
						Source:   "data",
						Type:     "volume",
						ReadOnly: true,
						Volume:   &types.ServiceVolumeVolume{NoCopy: true},
					},
				},
				Environment: types.MappingWithEquals{},
			},
		},
		Configs: map[string]types.ConfigObjConfig{
			"appconfig": {External: types.External{External: true}, Name: "appconfig"},
		},
		Secrets: map[string]types.SecretConfig{
			"super": {External: types.External{External: true}, Name: "super"},
		},
		Volumes: map[string]types.VolumeConfig{
			"data": {External: types.External{External: true}, Name: "data"},
		},
		Networks: map[string]types.NetworkConfig{
			"front": {
				External:   types.External{External: true},
				Name:       "front",
				Internal:   true,
				Attachable: true,
			},
		},
	}

	assert.Check(t, is.DeepEqual(expected, config))
}

func TestUnsupportedProperties(t *testing.T) {
	dict, err := ParseYAML([]byte(`
version: "3"
services:
  web:
    image: web
    build:
     context: ./web
    links:
      - bar
    pid: host
  db:
    image: db
    build:
     context: ./db
`))
	assert.NilError(t, err)

	configDetails := buildConfigDetails(dict, nil)

	_, err = Load(configDetails)
	assert.NilError(t, err)

	unsupported := GetUnsupportedProperties(dict)
	assert.Check(t, is.DeepEqual([]string{"build", "links", "pid"}, unsupported))
}

func TestDiscardEnvFileOption(t *testing.T) {
	dict, err := ParseYAML([]byte(`version: "3"
services:
  web:
    image: nginx
    env_file:
     - example1.env
     - example2.env
`))
	expectedEnvironmentMap := types.MappingWithEquals{
		"FOO": strPtr("foo_from_env_file"),
		"BAZ": strPtr("baz_from_env_file"),
		"BAR": strPtr("bar_from_env_file_2"), // Original value is overwritten by example2.env
		"QUX": strPtr("quz_from_env_file_2"),
	}
	assert.NilError(t, err)
	configDetails := buildConfigDetails(dict, nil)

	// Default behavior keeps the `env_file` entries
	configWithEnvFiles, err := Load(configDetails)
	assert.NilError(t, err)
	expected := types.StringList{"example1.env", "example2.env"}
	assert.Check(t, is.DeepEqual(expected, configWithEnvFiles.Services[0].EnvFile))
	assert.Check(t, is.DeepEqual(expectedEnvironmentMap, configWithEnvFiles.Services[0].Environment))

	// Custom behavior removes the `env_file` entries
	configWithoutEnvFiles, err := Load(configDetails, WithDiscardEnvFiles)
	assert.NilError(t, err)
	expected = types.StringList(nil)
	assert.Check(t, is.DeepEqual(expected, configWithoutEnvFiles.Services[0].EnvFile))
	assert.Check(t, is.DeepEqual(expectedEnvironmentMap, configWithoutEnvFiles.Services[0].Environment))
}

func TestBuildProperties(t *testing.T) {
	dict, err := ParseYAML([]byte(`
version: "3"
services:
  web:
    image: web
    build: .
    links:
      - bar
  db:
    image: db
    build:
     context: ./db
`))
	assert.NilError(t, err)
	configDetails := buildConfigDetails(dict, nil)
	_, err = Load(configDetails)
	assert.NilError(t, err)
}

func TestDeprecatedProperties(t *testing.T) {
	dict, err := ParseYAML([]byte(`
version: "3"
services:
  web:
    image: web
    container_name: web
  db:
    image: db
    container_name: db
    expose: ["5434"]
`))
	assert.NilError(t, err)

	configDetails := buildConfigDetails(dict, nil)

	_, err = Load(configDetails)
	assert.NilError(t, err)

	deprecated := GetDeprecatedProperties(dict)
	assert.Check(t, is.Len(deprecated, 2))
	assert.Check(t, is.Contains(deprecated, "container_name"))
	assert.Check(t, is.Contains(deprecated, "expose"))
}

func TestForbiddenProperties(t *testing.T) {
	_, err := loadYAML(`
version: "3"
services:
  foo:
    image: busybox
    volumes:
      - /data
    volume_driver: some-driver
  bar:
    extends:
      service: foo
`)

	assert.ErrorType(t, err, &ForbiddenPropertiesError{})

	props := err.(*ForbiddenPropertiesError).Properties
	assert.Check(t, is.Len(props, 2))
	assert.Check(t, is.Contains(props, "volume_driver"))
	assert.Check(t, is.Contains(props, "extends"))
}

func TestInvalidResource(t *testing.T) {
	_, err := loadYAML(`
        version: "3"
        services:
          foo:
            image: busybox
            deploy:
              resources:
                impossible:
                  x: 1
`)
	assert.Check(t, is.ErrorContains(err, "Additional property impossible is not allowed"))
}

func TestInvalidExternalAndDriverCombination(t *testing.T) {
	_, err := loadYAML(`
version: "3"
volumes:
  external_volume:
    external: true
    driver: foobar
`)

	assert.Check(t, is.ErrorContains(err, `conflicting parameters "external" and "driver" specified for volume`))
	assert.Check(t, is.ErrorContains(err, `external_volume`))
}

func TestInvalidExternalAndDirverOptsCombination(t *testing.T) {
	_, err := loadYAML(`
version: "3"
volumes:
  external_volume:
    external: true
    driver_opts:
      beep: boop
`)

	assert.Check(t, is.ErrorContains(err, `conflicting parameters "external" and "driver_opts" specified for volume`))
	assert.Check(t, is.ErrorContains(err, `external_volume`))
}

func TestInvalidExternalAndLabelsCombination(t *testing.T) {
	_, err := loadYAML(`
version: "3"
volumes:
  external_volume:
    external: true
    labels:
      - beep=boop
`)

	assert.Check(t, is.ErrorContains(err, `conflicting parameters "external" and "labels" specified for volume`))
	assert.Check(t, is.ErrorContains(err, `external_volume`))
}

func TestLoadVolumeInvalidExternalNameAndNameCombination(t *testing.T) {
	_, err := loadYAML(`
version: "3.4"
volumes:
  external_volume:
    name: user_specified_name
    external:
      name: external_name
`)

	assert.Check(t, is.ErrorContains(err, "volume.external.name and volume.name conflict; only use volume.name"))
	assert.Check(t, is.ErrorContains(err, `external_volume`))
}

func durationPtr(value time.Duration) *types.Duration {
	result := types.Duration(value)
	return &result
}

func uint64Ptr(value uint64) *uint64 {
	return &value
}

func uint32Ptr(value uint32) *uint32 {
	return &value
}

func TestFullExample(t *testing.T) {
	data, err := os.ReadFile("full-example.yml")
	assert.NilError(t, err)

	homeDir := "/home/foo"
	env := map[string]string{"HOME": homeDir, "QUX": "qux_from_environment"}
	config, err := loadYAMLWithEnv(string(data), env)
	assert.NilError(t, err)

	workingDir, err := os.Getwd()
	assert.NilError(t, err)

	expected := fullExampleConfig(workingDir, homeDir)

	assert.Check(t, is.DeepEqual(expected.Services, config.Services))
	assert.Check(t, is.DeepEqual(expected.Networks, config.Networks))
	assert.Check(t, is.DeepEqual(expected.Volumes, config.Volumes))
	assert.Check(t, is.DeepEqual(expected.Secrets, config.Secrets))
	assert.Check(t, is.DeepEqual(expected.Configs, config.Configs))
	assert.Check(t, is.DeepEqual(expected.Extras, config.Extras))
}

func TestLoadTmpfsVolume(t *testing.T) {
	config, err := loadYAML(`
version: "3.6"
services:
  tmpfs:
    image: nginx:latest
    volumes:
      - type: tmpfs
        target: /app
        tmpfs:
          size: 10000
`)
	assert.NilError(t, err)

	expected := types.ServiceVolumeConfig{
		Target: "/app",
		Type:   "tmpfs",
		Tmpfs: &types.ServiceVolumeTmpfs{
			Size: int64(10000),
		},
	}

	assert.Assert(t, is.Len(config.Services, 1))
	assert.Check(t, is.Len(config.Services[0].Volumes, 1))
	assert.Check(t, is.DeepEqual(expected, config.Services[0].Volumes[0]))
}

func TestLoadTmpfsVolumeAdditionalPropertyNotAllowed(t *testing.T) {
	_, err := loadYAML(`
version: "3.5"
services:
  tmpfs:
    image: nginx:latest
    volumes:
      - type: tmpfs
        target: /app
        tmpfs:
          size: 10000
`)
	assert.Check(t, is.ErrorContains(err, "services.tmpfs.volumes.0 Additional property tmpfs is not allowed"))
}

func TestLoadBindMountSourceMustNotBeEmpty(t *testing.T) {
	_, err := loadYAML(`
version: "3.5"
services:
  tmpfs:
    image: nginx:latest
    volumes:
      - type: bind
        target: /app
`)
	assert.Check(t, is.Error(err, `invalid mount config for type "bind": field Source must not be empty`))
}

func TestLoadBindMountSourceIsWindowsAbsolute(t *testing.T) {
	tests := []struct {
		doc      string
		yaml     string
		expected types.ServiceVolumeConfig
	}{
		{
			doc: "Z-drive lowercase",
			yaml: `
version: '3.3'

services:
  windows:
    image: mcr.microsoft.com/windows/servercore/iis:windowsservercore-ltsc2019
    volumes:
      - type: bind
        source: z:\
        target: c:\data
`,
			expected: types.ServiceVolumeConfig{Type: "bind", Source: `z:\`, Target: `c:\data`},
		},
		{
			doc: "Z-drive uppercase",
			yaml: `
version: '3.3'

services:
  windows:
    image: mcr.microsoft.com/windows/servercore/iis:windowsservercore-ltsc2019
    volumes:
      - type: bind
        source: Z:\
        target: C:\data
`,
			expected: types.ServiceVolumeConfig{Type: "bind", Source: `Z:\`, Target: `C:\data`},
		},
		{
			doc: "Z-drive subdirectory",
			yaml: `
version: '3.3'

services:
  windows:
    image: mcr.microsoft.com/windows/servercore/iis:windowsservercore-ltsc2019
    volumes:
      - type: bind
        source: Z:\some-dir
        target: C:\data
`,
			expected: types.ServiceVolumeConfig{Type: "bind", Source: `Z:\some-dir`, Target: `C:\data`},
		},
		{
			doc: "forward-slashes",
			yaml: `
version: '3.3'

services:
  app:
    image: app:latest
    volumes:
      - type: bind
        source: /z/some-dir
        target: /c/data
`,
			expected: types.ServiceVolumeConfig{Type: "bind", Source: `/z/some-dir`, Target: `/c/data`},
		},
	}

	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			config, err := loadYAML(tc.yaml)
			assert.NilError(t, err)
			assert.Check(t, is.Len(config.Services[0].Volumes, 1))
			assert.Check(t, is.DeepEqual(tc.expected, config.Services[0].Volumes[0]))
		})
	}
}

func TestLoadBindMountWithSource(t *testing.T) {
	config, err := loadYAML(`
version: "3.5"
services:
  bind:
    image: nginx:latest
    volumes:
      - type: bind
        target: /app
        source: "."
`)
	assert.NilError(t, err)

	workingDir, err := os.Getwd()
	assert.NilError(t, err)

	expected := types.ServiceVolumeConfig{
		Type:   "bind",
		Source: workingDir,
		Target: "/app",
	}

	assert.Assert(t, is.Len(config.Services, 1))
	assert.Check(t, is.Len(config.Services[0].Volumes, 1))
	assert.Check(t, is.DeepEqual(expected, config.Services[0].Volumes[0]))
}

func TestLoadTmpfsVolumeSizeCanBeZero(t *testing.T) {
	config, err := loadYAML(`
version: "3.6"
services:
  tmpfs:
    image: nginx:latest
    volumes:
      - type: tmpfs
        target: /app
        tmpfs:
          size: 0
`)
	assert.NilError(t, err)

	expected := types.ServiceVolumeConfig{
		Target: "/app",
		Type:   "tmpfs",
		Tmpfs:  &types.ServiceVolumeTmpfs{},
	}

	assert.Assert(t, is.Len(config.Services, 1))
	assert.Check(t, is.Len(config.Services[0].Volumes, 1))
	assert.Check(t, is.DeepEqual(expected, config.Services[0].Volumes[0]))
}

func TestLoadTmpfsVolumeSizeMustBeGTEQZero(t *testing.T) {
	_, err := loadYAML(`
version: "3.6"
services:
  tmpfs:
    image: nginx:latest
    volumes:
      - type: tmpfs
        target: /app
        tmpfs:
          size: -1
`)
	assert.ErrorContains(t, err, "services.tmpfs.volumes.0.tmpfs.size Must be greater than or equal to 0")
}

func TestLoadTmpfsVolumeSizeMustBeInteger(t *testing.T) {
	_, err := loadYAML(`
version: "3.6"
services:
  tmpfs:
    image: nginx:latest
    volumes:
      - type: tmpfs
        target: /app
        tmpfs:
          size: 0.0001
`)
	assert.ErrorContains(t, err, "services.tmpfs.volumes.0.tmpfs.size must be a integer")
}

func serviceSort(services []types.ServiceConfig) []types.ServiceConfig {
	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})
	return services
}

func TestLoadAttachableNetwork(t *testing.T) {
	config, err := loadYAML(`
version: "3.2"
networks:
  mynet1:
    driver: overlay
    attachable: true
  mynet2:
    driver: bridge
`)
	assert.NilError(t, err)

	expected := map[string]types.NetworkConfig{
		"mynet1": {
			Driver:     "overlay",
			Attachable: true,
		},
		"mynet2": {
			Driver:     "bridge",
			Attachable: false,
		},
	}

	assert.Check(t, is.DeepEqual(expected, config.Networks))
}

func TestLoadExpandedPortFormat(t *testing.T) {
	config, err := loadYAML(`
version: "3.2"
services:
  web:
    image: busybox
    ports:
      - "80-82:8080-8082"
      - "90-92:8090-8092/udp"
      - "85:8500"
      - 8600
      - protocol: udp
        target: 53
        published: 10053
      - mode: host
        target: 22
        published: 10022
`)
	assert.NilError(t, err)

	expected := samplePortsConfig
	assert.Check(t, is.Len(config.Services, 1))
	assert.Check(t, is.DeepEqual(expected, config.Services[0].Ports))
}

func TestLoadExpandedMountFormat(t *testing.T) {
	config, err := loadYAML(`
version: "3.2"
services:
  web:
    image: busybox
    volumes:
      - type: volume
        source: foo
        target: /target
        read_only: true
volumes:
  foo: {}
`)
	assert.NilError(t, err)

	expected := types.ServiceVolumeConfig{
		Type:     "volume",
		Source:   "foo",
		Target:   "/target",
		ReadOnly: true,
	}

	assert.Assert(t, is.Len(config.Services, 1))
	assert.Check(t, is.Len(config.Services[0].Volumes, 1))
	assert.Check(t, is.DeepEqual(expected, config.Services[0].Volumes[0]))
}

func TestLoadExtraHostsMap(t *testing.T) {
	config, err := loadYAML(`
version: "3.2"
services:
  web:
    image: busybox
    extra_hosts:
      "zulu": "162.242.195.82"
      "alpha": "50.31.209.229"
      "host.docker.internal": "host-gateway"
`)
	assert.NilError(t, err)

	expected := types.HostsList{
		"alpha:50.31.209.229",
		"host.docker.internal:host-gateway",
		"zulu:162.242.195.82",
	}

	assert.Assert(t, is.Len(config.Services, 1))
	assert.Check(t, is.DeepEqual(expected, config.Services[0].ExtraHosts))
}

func TestLoadExtraHostsList(t *testing.T) {
	config, err := loadYAML(`
version: "3.2"
services:
  web:
    image: busybox
    extra_hosts:
      - "zulu:162.242.195.82"
      - "alpha:50.31.209.229"
      - "zulu:ff02::1"
      - "host.docker.internal:host-gateway"
`)
	assert.NilError(t, err)

	expected := types.HostsList{
		"zulu:162.242.195.82",
		"alpha:50.31.209.229",
		"zulu:ff02::1",
		"host.docker.internal:host-gateway",
	}

	assert.Assert(t, is.Len(config.Services, 1))
	assert.Check(t, is.DeepEqual(expected, config.Services[0].ExtraHosts))
}

func TestLoadVolumesWarnOnDeprecatedExternalNameVersion34(t *testing.T) {
	buf, cleanup := patchLogrus()
	defer cleanup()

	source := map[string]interface{}{
		"foo": map[string]interface{}{
			"external": map[string]interface{}{
				"name": "oops",
			},
		},
	}
	vols, err := LoadVolumes(source, "3.4")
	assert.NilError(t, err)
	expected := map[string]types.VolumeConfig{
		"foo": {
			Name:     "oops",
			External: types.External{External: true},
		},
	}
	assert.Check(t, is.DeepEqual(expected, vols))
	assert.Check(t, is.Contains(buf.String(), "volume.external.name is deprecated"))
}

func patchLogrus() (*bytes.Buffer, func()) {
	buf := new(bytes.Buffer)
	out := logrus.StandardLogger().Out
	logrus.SetOutput(buf)
	return buf, func() { logrus.SetOutput(out) }
}

func TestLoadVolumesWarnOnDeprecatedExternalNameVersion33(t *testing.T) {
	buf, cleanup := patchLogrus()
	defer cleanup()

	source := map[string]interface{}{
		"foo": map[string]interface{}{
			"external": map[string]interface{}{
				"name": "oops",
			},
		},
	}
	vols, err := LoadVolumes(source, "3.3")
	assert.NilError(t, err)
	expected := map[string]types.VolumeConfig{
		"foo": {
			Name:     "oops",
			External: types.External{External: true},
		},
	}
	assert.Check(t, is.DeepEqual(expected, vols))
	assert.Check(t, is.Equal("", buf.String()))
}

func TestLoadV35(t *testing.T) {
	actual, err := loadYAML(`
version: "3.5"
services:
  foo:
    image: busybox
    isolation: process
configs:
  foo:
    name: fooqux
    external: true
  bar:
    name: barqux
    file: ./example1.env
secrets:
  foo:
    name: fooqux
    external: true
  bar:
    name: barqux
    file: ./full-example.yml
`)
	assert.NilError(t, err)
	assert.Check(t, is.Len(actual.Services, 1))
	assert.Check(t, is.Len(actual.Secrets, 2))
	assert.Check(t, is.Len(actual.Configs, 2))
	assert.Check(t, is.Equal("process", actual.Services[0].Isolation))
}

func TestLoadV35InvalidIsolation(t *testing.T) {
	// validation should be done only on the daemon side
	actual, err := loadYAML(`
version: "3.5"
services:
  foo:
    image: busybox
    isolation: invalid
configs:
  super:
    external: true
`)
	assert.NilError(t, err)
	assert.Assert(t, is.Len(actual.Services, 1))
	assert.Check(t, is.Equal("invalid", actual.Services[0].Isolation))
}

func TestLoadSecretInvalidExternalNameAndNameCombination(t *testing.T) {
	_, err := loadYAML(`
version: "3.5"
secrets:
  external_secret:
    name: user_specified_name
    external:
      name: external_name
`)

	assert.Check(t, is.ErrorContains(err, "secret.external.name and secret.name conflict; only use secret.name"))
	assert.Check(t, is.ErrorContains(err, "external_secret"))
}

func TestLoadSecretsWarnOnDeprecatedExternalNameVersion35(t *testing.T) {
	buf, cleanup := patchLogrus()
	defer cleanup()

	source := map[string]interface{}{
		"foo": map[string]interface{}{
			"external": map[string]interface{}{
				"name": "oops",
			},
		},
	}
	details := types.ConfigDetails{
		Version: "3.5",
	}
	s, err := LoadSecrets(source, details)
	assert.NilError(t, err)
	expected := map[string]types.SecretConfig{
		"foo": {
			Name:     "oops",
			External: types.External{External: true},
		},
	}
	assert.Check(t, is.DeepEqual(expected, s))
	assert.Check(t, is.Contains(buf.String(), "secret.external.name is deprecated"))
}

func TestLoadNetworksWarnOnDeprecatedExternalNameVersion35(t *testing.T) {
	buf, cleanup := patchLogrus()
	defer cleanup()

	source := map[string]interface{}{
		"foo": map[string]interface{}{
			"external": map[string]interface{}{
				"name": "oops",
			},
		},
	}
	nws, err := LoadNetworks(source, "3.5")
	assert.NilError(t, err)
	expected := map[string]types.NetworkConfig{
		"foo": {
			Name:     "oops",
			External: types.External{External: true},
		},
	}
	assert.Check(t, is.DeepEqual(expected, nws))
	assert.Check(t, is.Contains(buf.String(), "network.external.name is deprecated"))
}

func TestLoadNetworksWarnOnDeprecatedExternalNameVersion34(t *testing.T) {
	buf, cleanup := patchLogrus()
	defer cleanup()

	source := map[string]interface{}{
		"foo": map[string]interface{}{
			"external": map[string]interface{}{
				"name": "oops",
			},
		},
	}
	networks, err := LoadNetworks(source, "3.4")
	assert.NilError(t, err)
	expected := map[string]types.NetworkConfig{
		"foo": {
			Name:     "oops",
			External: types.External{External: true},
		},
	}
	assert.Check(t, is.DeepEqual(expected, networks))
	assert.Check(t, is.Equal("", buf.String()))
}

func TestLoadNetworkInvalidExternalNameAndNameCombination(t *testing.T) {
	_, err := loadYAML(`
version: "3.5"
networks:
  foo:
    name: user_specified_name
    external:
      name: external_name
`)

	assert.Check(t, is.ErrorContains(err, "network.external.name and network.name conflict; only use network.name"))
	assert.Check(t, is.ErrorContains(err, "foo"))
}

func TestLoadNetworkWithName(t *testing.T) {
	config, err := loadYAML(`
version: '3.5'
services:
  hello-world:
    image: redis:alpine
    networks:
      - network1
      - network3

networks:
  network1:
    name: network2
  network3:
`)
	assert.NilError(t, err)
	expected := &types.Config{
		Filename: "filename.yml",
		Version:  "3.5",
		Services: types.Services{
			{
				Name:  "hello-world",
				Image: "redis:alpine",
				Networks: map[string]*types.ServiceNetworkConfig{
					"network1": nil,
					"network3": nil,
				},
			},
		},
		Networks: map[string]types.NetworkConfig{
			"network1": {Name: "network2"},
			"network3": {},
		},
	}
	assert.Check(t, is.DeepEqual(expected, config, cmpopts.EquateEmpty()))
}

func TestLoadInit(t *testing.T) {
	booleanTrue := true
	booleanFalse := false

	testcases := []struct {
		doc  string
		yaml string
		init *bool
	}{
		{
			doc: "no init defined",
			yaml: `
version: '3.7'
services:
  foo:
    image: alpine`,
		},
		{
			doc: "has true init",
			yaml: `
version: '3.7'
services:
  foo:
    image: alpine
    init: true`,
			init: &booleanTrue,
		},
		{
			doc: "has false init",
			yaml: `
version: '3.7'
services:
  foo:
    image: alpine
    init: false`,
			init: &booleanFalse,
		},
	}
	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.doc, func(t *testing.T) {
			config, err := loadYAML(testcase.yaml)
			assert.NilError(t, err)
			assert.Check(t, is.Len(config.Services, 1))
			assert.Check(t, is.DeepEqual(testcase.init, config.Services[0].Init))
		})
	}
}

func TestLoadSysctls(t *testing.T) {
	config, err := loadYAML(`
version: "3.8"
services:
  web:
    image: busybox
    sysctls:
      - net.core.somaxconn=1024
      - net.ipv4.tcp_syncookies=0
      - testing.one.one=
      - testing.one.two
`)
	assert.NilError(t, err)

	expected := types.Mapping{
		"net.core.somaxconn":      "1024",
		"net.ipv4.tcp_syncookies": "0",
		"testing.one.one":         "",
		"testing.one.two":         "",
	}

	assert.Assert(t, is.Len(config.Services, 1))
	assert.Check(t, is.DeepEqual(expected, config.Services[0].Sysctls))

	config, err = loadYAML(`
version: "3.8"
services:
  web:
    image: busybox
    sysctls:
      net.core.somaxconn: 1024
      net.ipv4.tcp_syncookies: 0
      testing.one.one: ""
      testing.one.two:
`)
	assert.NilError(t, err)

	assert.Assert(t, is.Len(config.Services, 1))
	assert.Check(t, is.DeepEqual(expected, config.Services[0].Sysctls))
}

func TestTransform(t *testing.T) {
	source := []interface{}{
		"80-82:8080-8082",
		"90-92:8090-8092/udp",
		"85:8500",
		8600,
		map[string]interface{}{
			"protocol":  "udp",
			"target":    53,
			"published": 10053,
		},
		map[string]interface{}{
			"mode":      "host",
			"target":    22,
			"published": 10022,
		},
	}
	var ports []types.ServicePortConfig
	err := Transform(source, &ports)
	assert.NilError(t, err)

	assert.Check(t, is.DeepEqual(samplePortsConfig, ports))
}

func TestLoadTemplateDriver(t *testing.T) {
	config, err := loadYAML(`
version: '3.8'
services:
  hello-world:
    image: redis:alpine
    secrets:
      - secret
    configs:
      - config

configs:
  config:
    name: config
    external: true
    template_driver: config-driver

secrets:
  secret:
    name: secret
    external: true
    template_driver: secret-driver
`)
	assert.NilError(t, err)
	expected := &types.Config{
		Filename: "filename.yml",
		Version:  "3.8",
		Services: types.Services{
			{
				Name:  "hello-world",
				Image: "redis:alpine",
				Configs: []types.ServiceConfigObjConfig{
					{
						Source: "config",
					},
				},
				Secrets: []types.ServiceSecretConfig{
					{
						Source: "secret",
					},
				},
			},
		},
		Configs: map[string]types.ConfigObjConfig{
			"config": {
				Name:           "config",
				External:       types.External{External: true},
				TemplateDriver: "config-driver",
			},
		},
		Secrets: map[string]types.SecretConfig{
			"secret": {
				Name:           "secret",
				External:       types.External{External: true},
				TemplateDriver: "secret-driver",
			},
		},
	}
	assert.Check(t, is.DeepEqual(expected, config, cmpopts.EquateEmpty()))
}

func TestLoadSecretDriver(t *testing.T) {
	config, err := loadYAML(`
version: '3.8'
services:
  hello-world:
    image: redis:alpine
    secrets:
      - secret
    configs:
      - config

configs:
  config:
    name: config
    external: true

secrets:
  secret:
    name: secret
    driver: secret-bucket
    driver_opts:
      OptionA: value for driver option A
      OptionB: value for driver option B
`)
	assert.NilError(t, err)
	expected := &types.Config{
		Filename: "filename.yml",
		Version:  "3.8",
		Services: types.Services{
			{
				Name:  "hello-world",
				Image: "redis:alpine",
				Configs: []types.ServiceConfigObjConfig{
					{
						Source: "config",
					},
				},
				Secrets: []types.ServiceSecretConfig{
					{
						Source: "secret",
					},
				},
			},
		},
		Configs: map[string]types.ConfigObjConfig{
			"config": {
				Name:     "config",
				External: types.External{External: true},
			},
		},
		Secrets: map[string]types.SecretConfig{
			"secret": {
				Name:   "secret",
				Driver: "secret-bucket",
				DriverOpts: map[string]string{
					"OptionA": "value for driver option A",
					"OptionB": "value for driver option B",
				},
			},
		},
	}
	assert.Check(t, is.DeepEqual(expected, config, cmpopts.EquateEmpty()))
}
