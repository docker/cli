package loader

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/docker/cli/cli/compose/types"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func strPtr(val string) *string {
	return &val
}

var sampleConfig = types.Config{
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
	require.NoError(t, err)
	assert.Equal(t, sampleDict, dict)
}

func TestLoad(t *testing.T) {
	actual, err := Load(buildConfigDetails(sampleDict, nil))
	require.NoError(t, err)
	assert.Equal(t, serviceSort(sampleConfig.Services), serviceSort(actual.Services))
	assert.Equal(t, sampleConfig.Networks, actual.Networks)
	assert.Equal(t, sampleConfig.Volumes, actual.Volumes)
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
	require.NoError(t, err)
	assert.Len(t, actual.Services, 1)
	assert.Len(t, actual.Secrets, 1)
}

func TestLoadV33(t *testing.T) {
	actual, err := loadYAML(`
version: "3.3"
services:
  foo:
    image: busybox
    credential_spec:
      File: "/foo"
    configs: [super]
configs:
  super:
    external: true
`)
	require.NoError(t, err)
	require.Len(t, actual.Services, 1)
	assert.Equal(t, actual.Services[0].CredentialSpec.File, "/foo")
	require.Len(t, actual.Configs, 1)
}

func TestParseAndLoad(t *testing.T) {
	actual, err := loadYAML(sampleYAML)
	require.NoError(t, err)
	assert.Equal(t, serviceSort(sampleConfig.Services), serviceSort(actual.Services))
	assert.Equal(t, sampleConfig.Networks, actual.Networks)
	assert.Equal(t, sampleConfig.Volumes, actual.Volumes)
}

func TestInvalidTopLevelObjectType(t *testing.T) {
	_, err := loadYAML("1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Top-level object must be a mapping")

	_, err = loadYAML("\"hello\"")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Top-level object must be a mapping")

	_, err = loadYAML("[\"hello\"]")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Top-level object must be a mapping")
}

func TestNonStringKeys(t *testing.T) {
	_, err := loadYAML(`
version: "3"
123:
  foo:
    image: busybox
`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Non-string key at top level: 123")

	_, err = loadYAML(`
version: "3"
services:
  foo:
    image: busybox
  123:
    image: busybox
`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Non-string key in services: 123")

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
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Non-string key in networks.default.ipam.config[0]: 123")

	_, err = loadYAML(`
version: "3"
services:
  dict-env:
    image: busybox
    environment:
      1: FOO
`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Non-string key in services.dict-env.environment: 1")
}

func TestSupportedVersion(t *testing.T) {
	_, err := loadYAML(`
version: "3"
services:
  foo:
    image: busybox
`)
	require.NoError(t, err)

	_, err = loadYAML(`
version: "3.0"
services:
  foo:
    image: busybox
`)
	require.NoError(t, err)
}

func TestUnsupportedVersion(t *testing.T) {
	_, err := loadYAML(`
version: "2"
services:
  foo:
    image: busybox
`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version")

	_, err = loadYAML(`
version: "2.0"
services:
  foo:
    image: busybox
`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version")
}

func TestInvalidVersion(t *testing.T) {
	_, err := loadYAML(`
version: 3
services:
  foo:
    image: busybox
`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version must be a string")
}

func TestV1Unsupported(t *testing.T) {
	_, err := loadYAML(`
foo:
  image: busybox
`)
	assert.Error(t, err)
}

func TestNonMappingObject(t *testing.T) {
	_, err := loadYAML(`
version: "3"
services:
  - foo:
      image: busybox
`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "services must be a mapping")

	_, err = loadYAML(`
version: "3"
services:
  foo: busybox
`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "services.foo must be a mapping")

	_, err = loadYAML(`
version: "3"
networks:
  - default:
      driver: bridge
`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "networks must be a mapping")

	_, err = loadYAML(`
version: "3"
networks:
  default: bridge
`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "networks.default must be a mapping")

	_, err = loadYAML(`
version: "3"
volumes:
  - data:
      driver: local
`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "volumes must be a mapping")

	_, err = loadYAML(`
version: "3"
volumes:
  data: local
`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "volumes.data must be a mapping")
}

func TestNonStringImage(t *testing.T) {
	_, err := loadYAML(`
version: "3"
services:
  foo:
    image: ["busybox", "latest"]
`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "services.foo.image must be a string")
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
	assert.NoError(t, err)

	expected := types.MappingWithEquals{
		"FOO":  strPtr("1"),
		"BAR":  strPtr("2"),
		"BAZ":  strPtr("2.5"),
		"QUX":  strPtr("qux"),
		"QUUX": nil,
	}

	assert.Equal(t, 2, len(config.Services))

	for _, service := range config.Services {
		assert.Equal(t, expected, service.Environment)
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
	require.Error(t, err)
	assert.Contains(t, err.Error(), "services.dict-env.environment.FOO must be a string, number or null")
}

func TestInvalidEnvironmentObject(t *testing.T) {
	_, err := loadYAML(`
version: "3"
services:
  dict-env:
    image: busybox
    environment: "FOO=1"
`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "services.dict-env.environment must be a mapping")
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

	require.NoError(t, err)

	expectedLabels := types.Labels{
		"home1":       home,
		"home2":       home,
		"nonexistent": "",
		"default":     "default",
	}

	assert.Equal(t, expectedLabels, config.Services[0].Labels)
	assert.Equal(t, home, config.Networks["test"].Driver)
	assert.Equal(t, home, config.Volumes["test"].Driver)
}

func TestLoadWithInterpolationCastFull(t *testing.T) {
	dict, err := ParseYAML([]byte(`
version: "3.4"
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
      restart_policy:
        max_attempts: $theint
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
	require.NoError(t, err)
	env := map[string]string{
		"theint":   "555",
		"thefloat": "3.14",
		"thebool":  "true",
	}

	config, err := Load(buildConfigDetails(dict, env))
	require.NoError(t, err)
	expected := &types.Config{
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
					RestartPolicy: &types.RestartPolicy{
						MaxAttempts: uint64Ptr(555),
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

	assert.Equal(t, expected, config)
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
	require.NoError(t, err)

	configDetails := buildConfigDetails(dict, nil)

	_, err = Load(configDetails)
	require.NoError(t, err)

	unsupported := GetUnsupportedProperties(configDetails)
	assert.Equal(t, []string{"build", "links", "pid"}, unsupported)
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
	require.NoError(t, err)
	configDetails := buildConfigDetails(dict, nil)
	_, err = Load(configDetails)
	require.NoError(t, err)
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
	require.NoError(t, err)

	configDetails := buildConfigDetails(dict, nil)

	_, err = Load(configDetails)
	require.NoError(t, err)

	deprecated := GetDeprecatedProperties(configDetails)
	assert.Len(t, deprecated, 2)
	assert.Contains(t, deprecated, "container_name")
	assert.Contains(t, deprecated, "expose")
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

	require.Error(t, err)
	assert.IsType(t, &ForbiddenPropertiesError{}, err)
	fmt.Println(err)
	forbidden := err.(*ForbiddenPropertiesError).Properties

	assert.Len(t, forbidden, 2)
	assert.Contains(t, forbidden, "volume_driver")
	assert.Contains(t, forbidden, "extends")
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
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Additional property impossible is not allowed")
}

func TestInvalidExternalAndDriverCombination(t *testing.T) {
	_, err := loadYAML(`
version: "3"
volumes:
  external_volume:
    external: true
    driver: foobar
`)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "conflicting parameters \"external\" and \"driver\" specified for volume")
	assert.Contains(t, err.Error(), "external_volume")
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

	require.Error(t, err)
	assert.Contains(t, err.Error(), "conflicting parameters \"external\" and \"driver_opts\" specified for volume")
	assert.Contains(t, err.Error(), "external_volume")
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

	require.Error(t, err)
	assert.Contains(t, err.Error(), "conflicting parameters \"external\" and \"labels\" specified for volume")
	assert.Contains(t, err.Error(), "external_volume")
}

func TestLoadVolumeInvalidExternalNameAndNameCombination(t *testing.T) {
	_, err := loadYAML(`
version: "3.4"
volumes:
  external_volume:
    name: user_specified_name
    external:
      name:	external_name
`)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "volume.external.name and volume.name conflict; only use volume.name")
	assert.Contains(t, err.Error(), "external_volume")
}

func durationPtr(value time.Duration) *time.Duration {
	return &value
}

func uint64Ptr(value uint64) *uint64 {
	return &value
}

func uint32Ptr(value uint32) *uint32 {
	return &value
}

func TestFullExample(t *testing.T) {
	bytes, err := ioutil.ReadFile("full-example.yml")
	require.NoError(t, err)

	homeDir := "/home/foo"
	env := map[string]string{"HOME": homeDir, "QUX": "qux_from_environment"}
	config, err := loadYAMLWithEnv(string(bytes), env)
	require.NoError(t, err)

	workingDir, err := os.Getwd()
	require.NoError(t, err)

	stopGracePeriod := time.Duration(20 * time.Second)

	expectedServiceConfig := types.ServiceConfig{
		Name: "foo",

		Build: types.BuildConfig{
			Context:    "./dir",
			Dockerfile: "Dockerfile",
			Args:       map[string]*string{"foo": strPtr("bar")},
			Target:     "foo",
			Network:    "foo",
			CacheFrom:  []string{"foo", "bar"},
			Labels:     map[string]string{"FOO": "BAR"},
		},
		CapAdd:        []string{"ALL"},
		CapDrop:       []string{"NET_ADMIN", "SYS_ADMIN"},
		CgroupParent:  "m-executor-abcd",
		Command:       []string{"bundle", "exec", "thin", "-p", "3000"},
		ContainerName: "my-web-container",
		DependsOn:     []string{"db", "redis"},
		Deploy: types.DeployConfig{
			Mode:     "replicated",
			Replicas: uint64Ptr(6),
			Labels:   map[string]string{"FOO": "BAR"},
			UpdateConfig: &types.UpdateConfig{
				Parallelism:     uint64Ptr(3),
				Delay:           time.Duration(10 * time.Second),
				FailureAction:   "continue",
				Monitor:         time.Duration(60 * time.Second),
				MaxFailureRatio: 0.3,
				Order:           "start-first",
			},
			Resources: types.Resources{
				Limits: &types.Resource{
					NanoCPUs:    "0.001",
					MemoryBytes: 50 * 1024 * 1024,
				},
				Reservations: &types.Resource{
					NanoCPUs:    "0.0001",
					MemoryBytes: 20 * 1024 * 1024,
					GenericResources: []types.GenericResource{
						{
							DiscreteResourceSpec: &types.DiscreteGenericResource{
								Kind:  "gpu",
								Value: 2,
							},
						},
						{
							DiscreteResourceSpec: &types.DiscreteGenericResource{
								Kind:  "ssd",
								Value: 1,
							},
						},
					},
				},
			},
			RestartPolicy: &types.RestartPolicy{
				Condition:   "on-failure",
				Delay:       durationPtr(5 * time.Second),
				MaxAttempts: uint64Ptr(3),
				Window:      durationPtr(2 * time.Minute),
			},
			Placement: types.Placement{
				Constraints: []string{"node=foo"},
				Preferences: []types.PlacementPreferences{
					{
						Spread: "node.labels.az",
					},
				},
			},
			EndpointMode: "dnsrr",
		},
		Devices:    []string{"/dev/ttyUSB0:/dev/ttyUSB0"},
		DNS:        []string{"8.8.8.8", "9.9.9.9"},
		DNSSearch:  []string{"dc1.example.com", "dc2.example.com"},
		DomainName: "foo.com",
		Entrypoint: []string{"/code/entrypoint.sh", "-p", "3000"},
		Environment: map[string]*string{
			"FOO": strPtr("foo_from_env_file"),
			"BAR": strPtr("bar_from_env_file_2"),
			"BAZ": strPtr("baz_from_service_def"),
			"QUX": strPtr("qux_from_environment"),
		},
		EnvFile: []string{
			"./example1.env",
			"./example2.env",
		},
		Expose: []string{"3000", "8000"},
		ExternalLinks: []string{
			"redis_1",
			"project_db_1:mysql",
			"project_db_1:postgresql",
		},
		ExtraHosts: []string{
			"somehost:162.242.195.82",
			"otherhost:50.31.209.229",
		},
		HealthCheck: &types.HealthCheckConfig{
			Test:        types.HealthCheckTest([]string{"CMD-SHELL", "echo \"hello world\""}),
			Interval:    durationPtr(10 * time.Second),
			Timeout:     durationPtr(1 * time.Second),
			Retries:     uint64Ptr(5),
			StartPeriod: durationPtr(15 * time.Second),
		},
		Hostname: "foo",
		Image:    "redis",
		Ipc:      "host",
		Labels: map[string]string{
			"com.example.description": "Accounting webapp",
			"com.example.number":      "42",
			"com.example.empty-label": "",
		},
		Links: []string{
			"db",
			"db:database",
			"redis",
		},
		Logging: &types.LoggingConfig{
			Driver: "syslog",
			Options: map[string]string{
				"syslog-address": "tcp://192.168.0.42:123",
			},
		},
		MacAddress:  "02:42:ac:11:65:43",
		NetworkMode: "container:0cfeab0f748b9a743dc3da582046357c6ef497631c1a016d28d2bf9b4f899f7b",
		Networks: map[string]*types.ServiceNetworkConfig{
			"some-network": {
				Aliases:     []string{"alias1", "alias3"},
				Ipv4Address: "",
				Ipv6Address: "",
			},
			"other-network": {
				Ipv4Address: "172.16.238.10",
				Ipv6Address: "2001:3984:3989::10",
			},
			"other-other-network": nil,
		},
		Pid: "host",
		Ports: []types.ServicePortConfig{
			//"3000",
			{
				Mode:     "ingress",
				Target:   3000,
				Protocol: "tcp",
			},
			//"3000-3005",
			{
				Mode:     "ingress",
				Target:   3000,
				Protocol: "tcp",
			},
			{
				Mode:     "ingress",
				Target:   3001,
				Protocol: "tcp",
			},
			{
				Mode:     "ingress",
				Target:   3002,
				Protocol: "tcp",
			},
			{
				Mode:     "ingress",
				Target:   3003,
				Protocol: "tcp",
			},
			{
				Mode:     "ingress",
				Target:   3004,
				Protocol: "tcp",
			},
			{
				Mode:     "ingress",
				Target:   3005,
				Protocol: "tcp",
			},
			//"8000:8000",
			{
				Mode:      "ingress",
				Target:    8000,
				Published: 8000,
				Protocol:  "tcp",
			},
			//"9090-9091:8080-8081",
			{
				Mode:      "ingress",
				Target:    8080,
				Published: 9090,
				Protocol:  "tcp",
			},
			{
				Mode:      "ingress",
				Target:    8081,
				Published: 9091,
				Protocol:  "tcp",
			},
			//"49100:22",
			{
				Mode:      "ingress",
				Target:    22,
				Published: 49100,
				Protocol:  "tcp",
			},
			//"127.0.0.1:8001:8001",
			{
				Mode:      "ingress",
				Target:    8001,
				Published: 8001,
				Protocol:  "tcp",
			},
			//"127.0.0.1:5000-5010:5000-5010",
			{
				Mode:      "ingress",
				Target:    5000,
				Published: 5000,
				Protocol:  "tcp",
			},
			{
				Mode:      "ingress",
				Target:    5001,
				Published: 5001,
				Protocol:  "tcp",
			},
			{
				Mode:      "ingress",
				Target:    5002,
				Published: 5002,
				Protocol:  "tcp",
			},
			{
				Mode:      "ingress",
				Target:    5003,
				Published: 5003,
				Protocol:  "tcp",
			},
			{
				Mode:      "ingress",
				Target:    5004,
				Published: 5004,
				Protocol:  "tcp",
			},
			{
				Mode:      "ingress",
				Target:    5005,
				Published: 5005,
				Protocol:  "tcp",
			},
			{
				Mode:      "ingress",
				Target:    5006,
				Published: 5006,
				Protocol:  "tcp",
			},
			{
				Mode:      "ingress",
				Target:    5007,
				Published: 5007,
				Protocol:  "tcp",
			},
			{
				Mode:      "ingress",
				Target:    5008,
				Published: 5008,
				Protocol:  "tcp",
			},
			{
				Mode:      "ingress",
				Target:    5009,
				Published: 5009,
				Protocol:  "tcp",
			},
			{
				Mode:      "ingress",
				Target:    5010,
				Published: 5010,
				Protocol:  "tcp",
			},
		},
		Privileged: true,
		ReadOnly:   true,
		Restart:    "always",
		SecurityOpt: []string{
			"label=level:s0:c100,c200",
			"label=type:svirt_apache_t",
		},
		StdinOpen:       true,
		StopSignal:      "SIGUSR1",
		StopGracePeriod: &stopGracePeriod,
		Tmpfs:           []string{"/run", "/tmp"},
		Tty:             true,
		Ulimits: map[string]*types.UlimitsConfig{
			"nproc": {
				Single: 65535,
			},
			"nofile": {
				Soft: 20000,
				Hard: 40000,
			},
		},
		User: "someone",
		Volumes: []types.ServiceVolumeConfig{
			{Target: "/var/lib/mysql", Type: "volume"},
			{Source: "/opt/data", Target: "/var/lib/mysql", Type: "bind"},
			{Source: workingDir, Target: "/code", Type: "bind"},
			{Source: workingDir + "/static", Target: "/var/www/html", Type: "bind"},
			{Source: homeDir + "/configs", Target: "/etc/configs/", Type: "bind", ReadOnly: true},
			{Source: "datavolume", Target: "/var/lib/mysql", Type: "volume"},
			{Source: workingDir + "/opt", Target: "/opt", Consistency: "cached", Type: "bind"},
		},
		WorkingDir: "/code",
	}

	assert.Equal(t, []types.ServiceConfig{expectedServiceConfig}, config.Services)

	expectedNetworkConfig := map[string]types.NetworkConfig{
		"some-network": {},

		"other-network": {
			Driver: "overlay",
			DriverOpts: map[string]string{
				"foo": "bar",
				"baz": "1",
			},
			Ipam: types.IPAMConfig{
				Driver: "overlay",
				Config: []*types.IPAMPool{
					{Subnet: "172.16.238.0/24"},
					{Subnet: "2001:3984:3989::/64"},
				},
			},
		},

		"external-network": {
			Name:     "external-network",
			External: types.External{External: true},
		},

		"other-external-network": {
			Name:     "my-cool-network",
			External: types.External{External: true},
		},
	}

	assert.Equal(t, expectedNetworkConfig, config.Networks)

	expectedVolumeConfig := map[string]types.VolumeConfig{
		"some-volume": {},
		"other-volume": {
			Driver: "flocker",
			DriverOpts: map[string]string{
				"foo": "bar",
				"baz": "1",
			},
		},
		"another-volume": {
			Name:   "user_specified_name",
			Driver: "vsphere",
			DriverOpts: map[string]string{
				"foo": "bar",
				"baz": "1",
			},
		},
		"external-volume": {
			Name:     "external-volume",
			External: types.External{External: true},
		},
		"other-external-volume": {
			Name:     "my-cool-volume",
			External: types.External{External: true},
		},
		"external-volume3": {
			Name:     "this-is-volume3",
			External: types.External{External: true},
		},
	}

	assert.Equal(t, expectedVolumeConfig, config.Volumes)
}

func serviceSort(services []types.ServiceConfig) []types.ServiceConfig {
	sort.Sort(servicesByName(services))
	return services
}

type servicesByName []types.ServiceConfig

func (sbn servicesByName) Len() int           { return len(sbn) }
func (sbn servicesByName) Swap(i, j int)      { sbn[i], sbn[j] = sbn[j], sbn[i] }
func (sbn servicesByName) Less(i, j int) bool { return sbn[i].Name < sbn[j].Name }

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
	require.NoError(t, err)

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

	assert.Equal(t, expected, config.Networks)
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
	require.NoError(t, err)

	expected := []types.ServicePortConfig{
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

	assert.Len(t, config.Services, 1)
	assert.Equal(t, expected, config.Services[0].Ports)
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
	require.NoError(t, err)

	expected := types.ServiceVolumeConfig{
		Type:     "volume",
		Source:   "foo",
		Target:   "/target",
		ReadOnly: true,
	}

	require.Len(t, config.Services, 1)
	assert.Len(t, config.Services[0].Volumes, 1)
	assert.Equal(t, expected, config.Services[0].Volumes[0])
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
`)
	require.NoError(t, err)

	expected := types.HostsList{
		"alpha:50.31.209.229",
		"zulu:162.242.195.82",
	}

	require.Len(t, config.Services, 1)
	assert.Equal(t, expected, config.Services[0].ExtraHosts)
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
`)
	require.NoError(t, err)

	expected := types.HostsList{
		"zulu:162.242.195.82",
		"alpha:50.31.209.229",
		"zulu:ff02::1",
	}

	require.Len(t, config.Services, 1)
	assert.Equal(t, expected, config.Services[0].ExtraHosts)
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
	volumes, err := LoadVolumes(source, "3.4")
	require.NoError(t, err)
	expected := map[string]types.VolumeConfig{
		"foo": {
			Name:     "oops",
			External: types.External{External: true},
		},
	}
	assert.Equal(t, expected, volumes)
	assert.Contains(t, buf.String(), "volume.external.name is deprecated")

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
	volumes, err := LoadVolumes(source, "3.3")
	require.NoError(t, err)
	expected := map[string]types.VolumeConfig{
		"foo": {
			Name:     "oops",
			External: types.External{External: true},
		},
	}
	assert.Equal(t, expected, volumes)
	assert.Equal(t, "", buf.String())
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
	require.NoError(t, err)
	assert.Len(t, actual.Services, 1)
	assert.Len(t, actual.Secrets, 2)
	assert.Len(t, actual.Configs, 2)
	assert.Equal(t, "process", actual.Services[0].Isolation)
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
	require.NoError(t, err)
	require.Len(t, actual.Services, 1)
	assert.Equal(t, "invalid", actual.Services[0].Isolation)
}

func TestLoadSecretInvalidExternalNameAndNameCombination(t *testing.T) {
	_, err := loadYAML(`
version: "3.5"
secrets:
  external_secret:
    name: user_specified_name
    external:
      name:	external_name
`)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "secret.external.name and secret.name conflict; only use secret.name")
	assert.Contains(t, err.Error(), "external_secret")
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
	secrets, err := LoadSecrets(source, details)
	require.NoError(t, err)
	expected := map[string]types.SecretConfig{
		"foo": {
			Name:     "oops",
			External: types.External{External: true},
		},
	}
	assert.Equal(t, expected, secrets)
	assert.Contains(t, buf.String(), "secret.external.name is deprecated")
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
	networks, err := LoadNetworks(source, "3.5")
	require.NoError(t, err)
	expected := map[string]types.NetworkConfig{
		"foo": {
			Name:     "oops",
			External: types.External{External: true},
		},
	}
	assert.Equal(t, expected, networks)
	assert.Contains(t, buf.String(), "network.external.name is deprecated")

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
	require.NoError(t, err)
	expected := map[string]types.NetworkConfig{
		"foo": {
			Name:     "oops",
			External: types.External{External: true},
		},
	}
	assert.Equal(t, expected, networks)
	assert.Equal(t, "", buf.String())
}

func TestLoadNetworkInvalidExternalNameAndNameCombination(t *testing.T) {
	_, err := loadYAML(`
version: "3.5"
networks:
  foo:
    name: user_specified_name
    external:
      name:	external_name
`)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "network.external.name and network.name conflict; only use network.name")
	assert.Contains(t, err.Error(), "foo")
}
