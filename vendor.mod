module github.com/docker/cli

// 'vendor.mod' enables use of 'go mod vendor' to managed 'vendor/' directory.
// There is no 'go.mod' file, as that would imply opting in for all the rules
// around SemVer, which this repo cannot abide by as it uses CalVer.

go 1.16

require (
	github.com/Microsoft/go-winio v0.4.19 // indirect
	github.com/containerd/containerd v1.5.5
	github.com/coreos/etcd v3.3.25+incompatible // indirect
	github.com/creack/pty v1.1.11
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v20.10.7+incompatible
	github.com/docker/docker-credential-helpers v0.6.4
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-units v0.4.0
	github.com/docker/swarmkit v1.12.1-0.20210726173615-3629f50980f6
	github.com/fvbommel/sortorder v1.0.2
	github.com/gogo/protobuf v1.3.2
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-cmp v0.5.5
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	github.com/imdario/mergo v0.3.12
	github.com/mitchellh/mapstructure v1.3.2
	github.com/moby/buildkit v0.8.2-0.20210615162540-9f254e18360a
	github.com/moby/sys/signal v0.6.0
	github.com/moby/term v0.0.0-20210619224110-3f7ff695adc6
	github.com/morikuni/aec v1.0.0
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.1
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/theupdateframework/notary v0.7.1-0.20210315103452-bf96a202a09a
	github.com/tonistiigi/go-rosetta v0.0.0-20200727161949-f79598599c5d
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonschema v1.2.0
	golang.org/x/sys v0.0.0-20211025201205-69cdffdb9359
	golang.org/x/term v0.0.0-20201126162022-7de9c90e9dd1
	golang.org/x/text v0.3.4
	google.golang.org/grpc v1.38.0 // indirect
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools/v3 v3.0.3
)

replace (
	cloud.google.com/go => cloud.google.com/go v0.44.3
	github.com/docker/distribution => github.com/docker/distribution v2.7.1-0.20190205005809-0d3efadf0154+incompatible
	github.com/docker/docker => github.com/docker/docker v20.10.3-0.20210811141259-343665850e3a+incompatible // master (v21.xx-dev)
	github.com/docker/go => github.com/docker/go v1.5.1-1.0.20160303222718-d30aec9fd63c // contains a customized version of canonical/json and is used by Notary. The package is periodically rebased on current Go versions.
	github.com/evanphx/json-patch => gopkg.in/evanphx/json-patch.v4 v4.1.0
	github.com/gogo/googleapis => github.com/gogo/googleapis v1.3.2
	github.com/google/go-cmp => github.com/google/go-cmp v0.2.0
	github.com/google/gofuzz => github.com/google/gofuzz v0.0.0-20170612174753-24818f796faf
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.2.0
	github.com/imdario/mergo => github.com/imdario/mergo v0.3.12
	github.com/jaguilar/vt100 => github.com/tonistiigi/vt100 v0.0.0-20190402012908-ad4c4a574305
	github.com/json-iterator/go => github.com/json-iterator/go v1.1.10
	github.com/moby/buildkit => github.com/moby/buildkit v0.8.2-0.20210615162540-9f254e18360a // master (v0.9.0-dev)
	github.com/moby/sys/mount => github.com/moby/sys/mount v0.3.0
	github.com/moby/sys/mountinfo => github.com/moby/sys/mountinfo v0.5.0
	github.com/moby/sys/symlink => github.com/moby/sys/symlink v0.2.0
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v1.6.0
	github.com/prometheus/client_model => github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common => github.com/prometheus/common v0.9.1
	github.com/prometheus/procfs => github.com/prometheus/procfs v0.0.11
	github.com/theupdateframework/notary => github.com/theupdateframework/notary v0.7.1-0.20210315103452-bf96a202a09a
	golang.org/x/text => golang.org/x/text v0.3.3
	gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.2.8
	gotest.tools/v3 => gotest.tools/v3 v3.0.2
)
