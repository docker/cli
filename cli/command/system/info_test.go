package system

import (
	"encoding/base64"
	"net"
	"testing"
	"time"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/api/types/swarm"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/gotestyourself/gotestyourself/golden"
)

// helper function that base64 decodes a string and ignores the error
func base64Decode(val string) []byte {
	decoded, _ := base64.StdEncoding.DecodeString(val)
	return decoded
}

var sampleInfoNoSwarm = types.Info{
	ID:                "EKHL:QDUU:QZ7U:MKGD:VDXK:S27Q:GIPU:24B7:R7VT:DGN6:QCSF:2UBX",
	Containers:        0,
	ContainersRunning: 0,
	ContainersPaused:  0,
	ContainersStopped: 0,
	Images:            0,
	Driver:            "aufs",
	DriverStatus: [][2]string{
		{"Root Dir", "/var/lib/docker/aufs"},
		{"Backing Filesystem", "extfs"},
		{"Dirs", "0"},
		{"Dirperm1 Supported", "true"},
	},
	SystemStatus: nil,
	Plugins: types.PluginsInfo{
		Volume:        []string{"local"},
		Network:       []string{"bridge", "host", "macvlan", "null", "overlay"},
		Authorization: nil,
		Log:           []string{"awslogs", "fluentd", "gcplogs", "gelf", "journald", "json-file", "logentries", "splunk", "syslog"},
	},
	MemoryLimit:        true,
	SwapLimit:          true,
	KernelMemory:       true,
	CPUCfsPeriod:       true,
	CPUCfsQuota:        true,
	CPUShares:          true,
	CPUSet:             true,
	IPv4Forwarding:     true,
	BridgeNfIptables:   true,
	BridgeNfIP6tables:  true,
	Debug:              true,
	NFd:                33,
	OomKillDisable:     true,
	NGoroutines:        135,
	SystemTime:         "2017-08-24T17:44:34.077811894Z",
	LoggingDriver:      "json-file",
	CgroupDriver:       "cgroupfs",
	NEventsListener:    0,
	KernelVersion:      "4.4.0-87-generic",
	OperatingSystem:    "Ubuntu 16.04.3 LTS",
	OSType:             "linux",
	Architecture:       "x86_64",
	IndexServerAddress: "https://index.docker.io/v1/",
	RegistryConfig: &registry.ServiceConfig{
		AllowNondistributableArtifactsCIDRs:     nil,
		AllowNondistributableArtifactsHostnames: nil,
		InsecureRegistryCIDRs: []*registry.NetIPNet{
			{
				IP:   net.ParseIP("127.0.0.0"),
				Mask: net.IPv4Mask(255, 0, 0, 0),
			},
		},
		IndexConfigs: map[string]*registry.IndexInfo{
			"docker.io": {
				Name:     "docker.io",
				Mirrors:  nil,
				Secure:   true,
				Official: true,
			},
		},
		Mirrors: nil,
	},
	NCPU:              2,
	MemTotal:          2097356800,
	DockerRootDir:     "/var/lib/docker",
	HTTPProxy:         "",
	HTTPSProxy:        "",
	NoProxy:           "",
	Name:              "system-sample",
	Labels:            []string{"provider=digitalocean"},
	ExperimentalBuild: false,
	ServerVersion:     "17.06.1-ce",
	ClusterStore:      "",
	ClusterAdvertise:  "",
	Runtimes: map[string]types.Runtime{
		"runc": {
			Path: "docker-runc",
			Args: nil,
		},
	},
	DefaultRuntime:     "runc",
	Swarm:              swarm.Info{LocalNodeState: "inactive"},
	LiveRestoreEnabled: false,
	Isolation:          "",
	InitBinary:         "docker-init",
	ContainerdCommit: types.Commit{
		ID:       "6e23458c129b551d5c9871e5174f6b1b7f6d1170",
		Expected: "6e23458c129b551d5c9871e5174f6b1b7f6d1170",
	},
	RuncCommit: types.Commit{
		ID:       "810190ceaa507aa2727d7ae6f4790c76ec150bd2",
		Expected: "810190ceaa507aa2727d7ae6f4790c76ec150bd2",
	},
	InitCommit: types.Commit{
		ID:       "949e6fa",
		Expected: "949e6fa",
	},
	SecurityOptions: []string{"name=apparmor", "name=seccomp,profile=default"},
}

var sampleSwarmInfo = swarm.Info{
	NodeID:           "qo2dfdig9mmxqkawulggepdih",
	NodeAddr:         "165.227.107.89",
	LocalNodeState:   "active",
	ControlAvailable: true,
	Error:            "",
	RemoteManagers: []swarm.Peer{
		{
			NodeID: "qo2dfdig9mmxqkawulggepdih",
			Addr:   "165.227.107.89:2377",
		},
	},
	Nodes:    1,
	Managers: 1,
	Cluster: &swarm.ClusterInfo{
		ID: "9vs5ygs0gguyyec4iqf2314c0",
		Meta: swarm.Meta{
			Version:   swarm.Version{Index: 11},
			CreatedAt: time.Date(2017, 8, 24, 17, 34, 19, 278062352, time.UTC),
			UpdatedAt: time.Date(2017, 8, 24, 17, 34, 42, 398815481, time.UTC),
		},
		Spec: swarm.Spec{
			Annotations: swarm.Annotations{
				Name:   "default",
				Labels: nil,
			},
			Orchestration: swarm.OrchestrationConfig{
				TaskHistoryRetentionLimit: &[]int64{5}[0],
			},
			Raft: swarm.RaftConfig{
				SnapshotInterval:           10000,
				KeepOldSnapshots:           &[]uint64{0}[0],
				LogEntriesForSlowFollowers: 500,
				ElectionTick:               3,
				HeartbeatTick:              1,
			},
			Dispatcher: swarm.DispatcherConfig{
				HeartbeatPeriod: 5000000000,
			},
			CAConfig: swarm.CAConfig{
				NodeCertExpiry: 7776000000000000,
			},
			TaskDefaults: swarm.TaskDefaults{},
			EncryptionConfig: swarm.EncryptionConfig{
				AutoLockManagers: true,
			},
		},
		TLSInfo: swarm.TLSInfo{
			TrustRoot: `
-----BEGIN CERTIFICATE-----
MIIBajCCARCgAwIBAgIUaFCW5xsq8eyiJ+Pmcv3MCflMLnMwCgYIKoZIzj0EAwIw
EzERMA8GA1UEAxMIc3dhcm0tY2EwHhcNMTcwODI0MTcyOTAwWhcNMzcwODE5MTcy
OTAwWjATMREwDwYDVQQDEwhzd2FybS1jYTBZMBMGByqGSM49AgEGCCqGSM49AwEH
A0IABDy7NebyUJyUjWJDBUdnZoV6GBxEGKO4TZPNDwnxDxJcUdLVaB7WGa4/DLrW
UfsVgh1JGik2VTiLuTMA1tLlNPOjQjBAMA4GA1UdDwEB/wQEAwIBBjAPBgNVHRMB
Af8EBTADAQH/MB0GA1UdDgQWBBQl16XFtaaXiUAwEuJptJlDjfKskDAKBggqhkjO
PQQDAgNIADBFAiEAo9fTQNM5DP9bHVcTJYfl2Cay1bFu1E+lnpmN+EYJfeACIGKH
1pCUkZ+D0IB6CiEZGWSHyLuXPM1rlP+I5KuS7sB8
-----END CERTIFICATE-----
`,
			CertIssuerSubject: base64Decode("MBMxETAPBgNVBAMTCHN3YXJtLWNh"),
			CertIssuerPublicKey: base64Decode(
				"MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEPLs15vJQnJSNYkMFR2dmhXoYHEQYo7hNk80PCfEPElxR0tVoHtYZrj8MutZR+xWCHUkaKTZVOIu5MwDW0uU08w=="),
		},
		RootRotationInProgress: false,
	},
}

func TestPrettyPrintInfo(t *testing.T) {
	infoWithSwarm := sampleInfoNoSwarm
	infoWithSwarm.Swarm = sampleSwarmInfo

	infoWithWarningsLinux := sampleInfoNoSwarm
	infoWithWarningsLinux.MemoryLimit = false
	infoWithWarningsLinux.SwapLimit = false
	infoWithWarningsLinux.KernelMemory = false
	infoWithWarningsLinux.OomKillDisable = false
	infoWithWarningsLinux.CPUCfsQuota = false
	infoWithWarningsLinux.CPUCfsPeriod = false
	infoWithWarningsLinux.CPUShares = false
	infoWithWarningsLinux.CPUSet = false
	infoWithWarningsLinux.IPv4Forwarding = false
	infoWithWarningsLinux.BridgeNfIptables = false
	infoWithWarningsLinux.BridgeNfIP6tables = false

	for _, tc := range []struct {
		dockerInfo     types.Info
		expectedGolden string
		warningsGolden string
	}{
		{
			dockerInfo:     sampleInfoNoSwarm,
			expectedGolden: "docker-info-no-swarm",
		},
		{
			dockerInfo:     infoWithSwarm,
			expectedGolden: "docker-info-with-swarm",
		},
		{
			dockerInfo:     infoWithWarningsLinux,
			expectedGolden: "docker-info-no-swarm",
			warningsGolden: "docker-info-warnings",
		},
	} {
		cli := test.NewFakeCli(&fakeClient{})
		assert.NilError(t, prettyPrintInfo(cli, tc.dockerInfo))
		golden.Assert(t, cli.OutBuffer().String(), tc.expectedGolden+".golden")
		if tc.warningsGolden != "" {
			golden.Assert(t, cli.ErrBuffer().String(), tc.warningsGolden+".golden")
		} else {
			assert.Check(t, is.Equal("", cli.ErrBuffer().String()))
		}
	}
}
