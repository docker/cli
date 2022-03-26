module github.com/docker/cli

// 'vendor.mod' enables use of 'go mod vendor' to managed 'vendor/' directory.
// There is no 'go.mod' file, as that would imply opting in for all the rules
// around SemVer, which this repo cannot abide by as it uses CalVer.

go 1.17

require (
	github.com/containerd/containerd v1.6.2
	github.com/creack/pty v1.1.11
	github.com/docker/distribution v2.8.1+incompatible
	github.com/docker/docker v20.10.7+incompatible // see "replace" for the actual version
	github.com/docker/docker-credential-helpers v0.6.4
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-units v0.4.0
	github.com/docker/swarmkit v1.12.1-0.20220307221335-616e8db4c3b0
	github.com/fvbommel/sortorder v1.0.2
	github.com/gogo/protobuf v1.3.2
	github.com/google/go-cmp v0.5.7
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	github.com/imdario/mergo v0.3.12
	github.com/mitchellh/mapstructure v1.3.2
	github.com/moby/buildkit v0.10.0
	github.com/moby/sys/signal v0.7.0
	github.com/moby/term v0.0.0-20210619224110-3f7ff695adc6
	github.com/morikuni/aec v1.0.0
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.2
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/theupdateframework/notary v0.7.1-0.20210315103452-bf96a202a09a
	github.com/tonistiigi/go-rosetta v0.0.0-20200727161949-f79598599c5d
	github.com/xeipuuv/gojsonschema v1.2.0
	golang.org/x/sys v0.0.0-20220114195835-da31bd327af9
	golang.org/x/term v0.0.0-20210615171337-6886f2dfbf5b
	golang.org/x/text v0.3.7
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools/v3 v3.1.0
)

require (
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/Microsoft/go-winio v0.5.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/docker/go v1.5.1-1.0.20160303222718-d30aec9fd63c // indirect
	github.com/docker/go-events v0.0.0-20190806004212-e31b211e4f1c // indirect
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/gorilla/mux v1.8.0 // indirect; updated to v1.8.0 to get rid of old compatibility for "context"
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/klauspost/compress v1.15.0 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/miekg/pkcs11 v1.0.3 // indirect
	github.com/moby/sys/symlink v0.2.0 // indirect
	github.com/opencontainers/runc v1.1.0 // indirect
	github.com/prometheus/client_golang v1.12.1 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.32.1 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	go.etcd.io/etcd/raft/v3 v3.5.2 // indirect
	golang.org/x/crypto v0.0.0-20220315160706-3147a52a75dd // indirect
	golang.org/x/net v0.0.0-20211216030914-fe4d6282115f // indirect
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/genproto v0.0.0-20211208223120-3a66f561d7aa // indirect
	google.golang.org/grpc v1.44.0 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
)

replace (
	github.com/docker/docker => github.com/docker/docker v20.10.3-0.20220309172631-83b51522df43+incompatible // master (v21.xx-dev)
	github.com/gogo/googleapis => github.com/gogo/googleapis v1.3.2
)
