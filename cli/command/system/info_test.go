package system

import (
	"encoding/base64"
	"errors"
	"net/netip"
	"testing"
	"time"

	pluginmanager "github.com/docker/cli/cli-plugins/manager"
	"github.com/docker/cli/cli-plugins/metadata"
	"github.com/docker/cli/internal/test"
	registrytypes "github.com/moby/moby/api/types/registry"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/api/types/system"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/golden"
)

// helper function that base64 decodes a string and ignores the error
func base64Decode(val string) []byte {
	decoded, _ := base64.StdEncoding.DecodeString(val)
	return decoded
}

const sampleID = "EKHL:QDUU:QZ7U:MKGD:VDXK:S27Q:GIPU:24B7:R7VT:DGN6:QCSF:2UBX"

var sampleInfoNoSwarm = system.Info{
	ID:                sampleID,
	Containers:        0,
	ContainersRunning: 0,
	ContainersPaused:  0,
	ContainersStopped: 0,
	Images:            0,
	Driver:            "overlay2",
	DriverStatus: [][2]string{
		{"Backing Filesystem", "extfs"},
		{"Supports d_type", "true"},
		{"Using metacopy", "false"},
		{"Native Overlay Diff", "true"},
	},
	SystemStatus: nil,
	Plugins: system.PluginsInfo{
		Volume:        []string{"local"},
		Network:       []string{"bridge", "host", "macvlan", "null", "overlay"},
		Authorization: nil,
		Log:           []string{"awslogs", "fluentd", "gcplogs", "gelf", "journald", "json-file", "splunk", "syslog"},
	},
	MemoryLimit:        true,
	SwapLimit:          true,
	CPUCfsPeriod:       true,
	CPUCfsQuota:        true,
	CPUShares:          true,
	CPUSet:             true,
	IPv4Forwarding:     true,
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
	OSVersion:          "",
	OSType:             "linux",
	Architecture:       "x86_64",
	IndexServerAddress: "https://index.docker.io/v1/",
	RegistryConfig: &registrytypes.ServiceConfig{
		InsecureRegistryCIDRs: []netip.Prefix{
			netip.MustParsePrefix("127.0.0.0/8"),
		},
		IndexConfigs: map[string]*registrytypes.IndexInfo{
			"docker.io": {
				Name:     "docker.io",
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
	Runtimes: map[string]system.RuntimeWithStatus{
		"runc": {
			Runtime: system.Runtime{
				Path: "docker-runc",
				Args: nil,
			},
		},
	},
	DefaultRuntime:     "runc",
	Swarm:              swarm.Info{LocalNodeState: "inactive"},
	LiveRestoreEnabled: false,
	Isolation:          "",
	InitBinary:         "docker-init",
	ContainerdCommit: system.Commit{
		ID: "6e23458c129b551d5c9871e5174f6b1b7f6d1170",
	},
	RuncCommit: system.Commit{
		ID: "810190ceaa507aa2727d7ae6f4790c76ec150bd2",
	},
	InitCommit: system.Commit{
		ID: "949e6fa",
	},
	SecurityOptions: []string{"name=apparmor", "name=seccomp,profile=default"},
	DefaultAddressPools: []system.NetworkAddressPool{
		{
			Base: netip.MustParsePrefix("10.123.0.0/16"),
			Size: 24,
		},
	},
	FirewallBackend: &system.FirewallInfo{
		Driver: "nftables+firewalld",
		Info: [][2]string{
			{"ReloadedAt", "2025-07-16T16:59:14Z"},
		},
	},
	NRI: &system.NRIInfo{
		Info: [][2]string{
			{"plugin-path", "/usr/libexec/docker/nri-plugins"},
			{"plugin-config-path", "/etc/docker/nri/conf.d"},
		},
	},
	CDISpecDirs: []string{"/etc/cdi", "/var/run/cdi"},
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

var samplePluginsInfo = []pluginmanager.Plugin{
	{
		Name: "goodplugin",
		Path: "/path/to/docker-goodplugin",
		Metadata: metadata.Metadata{
			SchemaVersion:    "0.1.0",
			ShortDescription: "unit test is good",
			Vendor:           "ACME Corp",
			Version:          "0.1.0",
		},
	},
	{
		Name: "unversionedplugin",
		Path: "/path/to/docker-unversionedplugin",
		Metadata: metadata.Metadata{
			SchemaVersion:    "0.1.0",
			ShortDescription: "this plugin has no version",
			Vendor:           "ACME Corp",
		},
	},
	{
		Name: "badplugin",
		Path: "/path/to/docker-badplugin",
		Err:  errors.New("something wrong"),
	},
}

func TestPrettyPrintInfo(t *testing.T) {
	infoWithSwarm := sampleInfoNoSwarm
	infoWithSwarm.Swarm = sampleSwarmInfo

	infoWithWarningsLinux := sampleInfoNoSwarm
	infoWithWarningsLinux.MemoryLimit = false
	infoWithWarningsLinux.SwapLimit = false
	infoWithWarningsLinux.OomKillDisable = false
	infoWithWarningsLinux.CPUCfsQuota = false
	infoWithWarningsLinux.CPUCfsPeriod = false
	infoWithWarningsLinux.CPUShares = false
	infoWithWarningsLinux.CPUSet = false
	infoWithWarningsLinux.IPv4Forwarding = false

	sampleInfoDaemonWarnings := sampleInfoNoSwarm
	sampleInfoDaemonWarnings.Warnings = []string{
		"WARNING: No memory limit support",
		"WARNING: No swap limit support",
		"WARNING: No oom kill disable support",
		"WARNING: No cpu cfs quota support",
		"WARNING: No cpu cfs period support",
		"WARNING: No cpu shares support",
		"WARNING: No cpuset support",
		"WARNING: IPv4 forwarding is disabled",
	}

	sampleInfoBadSecurity := sampleInfoNoSwarm
	sampleInfoBadSecurity.SecurityOptions = []string{"foo="}

	sampleInfoLabelsNil := sampleInfoNoSwarm
	sampleInfoLabelsNil.Labels = nil
	sampleInfoLabelsEmpty := sampleInfoNoSwarm
	sampleInfoLabelsEmpty.Labels = []string{}

	sampleInfoWithDevices := sampleInfoNoSwarm
	sampleInfoWithDevices.DiscoveredDevices = []system.DeviceInfo{
		{Source: "cdi", ID: "com.example.device1"},
		{Source: "cdi", ID: "nvidia.com/gpu=gpu0"},
	}

	for _, tc := range []struct {
		doc        string
		dockerInfo dockerInfo

		prettyGolden   string
		warningsGolden string
		jsonGolden     string
		expectedError  string
	}{
		{
			doc: "info without swarm",
			dockerInfo: dockerInfo{
				Info: &sampleInfoNoSwarm,
				ClientInfo: &clientInfo{
					clientVersion: clientVersion{
						Platform: &platformInfo{Name: "Docker Engine - Community"},
						Version:  "24.0.0",
						Context:  "default",
					},
					Debug: true,
				},
			},
			prettyGolden: "docker-info-no-swarm",
			jsonGolden:   "docker-info-no-swarm",
		},
		{
			doc: "info with plugins",
			dockerInfo: dockerInfo{
				Info: &sampleInfoNoSwarm,
				ClientInfo: &clientInfo{
					clientVersion: clientVersion{Context: "default"},
					Plugins:       samplePluginsInfo,
				},
			},
			prettyGolden:   "docker-info-plugins",
			jsonGolden:     "docker-info-plugins",
			warningsGolden: "docker-info-plugins-warnings",
		},
		{
			doc: "info with nil labels",
			dockerInfo: dockerInfo{
				Info:       &sampleInfoLabelsNil,
				ClientInfo: &clientInfo{clientVersion: clientVersion{Context: "default"}},
			},
			prettyGolden: "docker-info-with-labels-nil",
		},
		{
			doc: "info with empty labels",
			dockerInfo: dockerInfo{
				Info:       &sampleInfoLabelsEmpty,
				ClientInfo: &clientInfo{clientVersion: clientVersion{Context: "default"}},
			},
			prettyGolden: "docker-info-with-labels-empty",
		},
		{
			doc: "info with swarm",
			dockerInfo: dockerInfo{
				Info: &infoWithSwarm,
				ClientInfo: &clientInfo{
					clientVersion: clientVersion{Context: "default"},
					Debug:         false,
				},
			},
			prettyGolden: "docker-info-with-swarm",
			jsonGolden:   "docker-info-with-swarm",
		},
		{
			doc: "info with daemon warnings",
			dockerInfo: dockerInfo{
				Info: &sampleInfoDaemonWarnings,
				ClientInfo: &clientInfo{
					clientVersion: clientVersion{
						Platform: &platformInfo{Name: "Docker Engine - Community"},
						Version:  "24.0.0",
						Context:  "default",
					},
					Debug: true,
				},
			},
			prettyGolden:   "docker-info-no-swarm",
			warningsGolden: "docker-info-warnings",
			jsonGolden:     "docker-info-daemon-warnings",
		},
		{
			doc: "errors for both",
			dockerInfo: dockerInfo{
				ServerErrors: []string{"a server error occurred"},
				ClientErrors: []string{"a client error occurred"},
			},
			prettyGolden:   "docker-info-errors",
			jsonGolden:     "docker-info-errors",
			warningsGolden: "docker-info-errors-stderr",
			expectedError:  "errors pretty printing info",
		},
		{
			doc: "bad security info",
			dockerInfo: dockerInfo{
				Info:         &sampleInfoBadSecurity,
				ServerErrors: []string{"a server error occurred"},
				ClientInfo:   &clientInfo{Debug: false},
			},
			prettyGolden:   "docker-info-badsec",
			jsonGolden:     "docker-info-badsec",
			warningsGolden: "docker-info-badsec-stderr",
			expectedError:  "errors pretty printing info",
		},
		{
			doc: "info with devices",
			dockerInfo: dockerInfo{
				Info: &sampleInfoWithDevices,
			},
			prettyGolden: "docker-info-with-devices",
			jsonGolden:   "docker-info-with-devices",
		},
	} {
		t.Run(tc.doc, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{})
			err := prettyPrintInfo(cli, tc.dockerInfo)
			if tc.expectedError == "" {
				assert.NilError(t, err)
			} else {
				assert.Error(t, err, tc.expectedError)
			}
			golden.Assert(t, cli.OutBuffer().String(), tc.prettyGolden+".golden")
			if tc.warningsGolden != "" {
				golden.Assert(t, cli.ErrBuffer().String(), tc.warningsGolden+".golden")
			} else {
				assert.Check(t, is.Equal("", cli.ErrBuffer().String()))
			}

			if tc.jsonGolden != "" {
				cli = test.NewFakeCli(&fakeClient{})
				assert.NilError(t, formatInfo(cli.Out(), tc.dockerInfo, "{{json .}}"))
				golden.Assert(t, cli.OutBuffer().String(), tc.jsonGolden+".json.golden")
				assert.Check(t, is.Equal("", cli.ErrBuffer().String()))

				cli = test.NewFakeCli(&fakeClient{})
				assert.NilError(t, formatInfo(cli.Out(), tc.dockerInfo, "json"))
				golden.Assert(t, cli.OutBuffer().String(), tc.jsonGolden+".json.golden")
				assert.Check(t, is.Equal("", cli.ErrBuffer().String()))
			}
		})
	}
}

func BenchmarkPrettyPrintInfo(b *testing.B) {
	infoWithSwarm := sampleInfoNoSwarm
	infoWithSwarm.Swarm = sampleSwarmInfo

	info := dockerInfo{
		Info: &infoWithSwarm,
		ClientInfo: &clientInfo{
			clientVersion: clientVersion{
				Platform: &platformInfo{Name: "Docker Engine - Community"},
				Version:  "24.0.0",
				Context:  "default",
			},
			Debug: true,
		},
	}
	cli := test.NewFakeCli(&fakeClient{})

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = prettyPrintInfo(cli, info)
		cli.ResetOutputBuffers()
	}
}

func TestFormatInfo(t *testing.T) {
	for _, tc := range []struct {
		doc           string
		template      string
		expectedError string
		expectedOut   string
	}{
		{
			doc:         "basic",
			template:    "{{.ID}}",
			expectedOut: sampleID + "\n",
		},
		{
			doc:           "syntax",
			template:      "{{}",
			expectedError: `template parsing error: template: :1: unexpected "}" in command`,
		},
		{
			doc:           "syntax",
			template:      "{{.badString}}",
			expectedError: `template: :1:2: executing "" at <.badString>: can't evaluate field badString in type system.dockerInfo`,
		},
	} {
		t.Run(tc.doc, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{})
			info := dockerInfo{
				Info:       &sampleInfoNoSwarm,
				ClientInfo: &clientInfo{Debug: true},
			}
			err := formatInfo(cli.Out(), info, tc.template)
			switch {
			case tc.expectedOut != "":
				assert.NilError(t, err)
				assert.Equal(t, cli.OutBuffer().String(), tc.expectedOut)
			case tc.expectedError != "":
				assert.Error(t, err, tc.expectedError)
			default:
				t.Fatal("test expected to neither pass nor fail")
			}
		})
	}
}

func TestNeedsServerInfo(t *testing.T) {
	tests := []struct {
		doc      string
		template string
		expected bool
	}{
		{
			doc:      "no template",
			template: "",
			expected: true,
		},
		{
			doc:      "JSON",
			template: "json",
			expected: true,
		},
		{
			doc:      "JSON (all fields)",
			template: "{{json .}}",
			expected: true,
		},
		{
			doc:      "JSON (Server ID)",
			template: "{{json .ID}}",
			expected: true,
		},
		{
			doc:      "ClientInfo",
			template: "{{json .ClientInfo}}",
			expected: false,
		},
		{
			doc:      "JSON ClientInfo",
			template: "{{json .ClientInfo}}",
			expected: false,
		},
		{
			doc:      "JSON (Active context)",
			template: "{{json .ClientInfo.Context}}",
			expected: false,
		},
	}

	inf := dockerInfo{ClientInfo: &clientInfo{}}
	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			assert.Equal(t, needsServerInfo(tc.template, inf), tc.expected)
		})
	}
}
