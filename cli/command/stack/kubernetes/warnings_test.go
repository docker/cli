package kubernetes

import (
	"bytes"
	"testing"
	"time"

	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/gotestyourself/gotestyourself/golden"
)

func TestWarnings(t *testing.T) {
	duration := 5 * time.Second
	attempts := uint64(3)
	config := &composetypes.Config{
		Version: "3.4",
		Services: []composetypes.ServiceConfig{
			{
				Name: "front",
				Build: composetypes.BuildConfig{
					Context: "ignored",
				},
				ContainerName:  "ignored",
				CgroupParent:   "ignored",
				CredentialSpec: composetypes.CredentialSpecConfig{File: "ignored"},
				DependsOn:      []string{"ignored"},
				Deploy: composetypes.DeployConfig{
					UpdateConfig: &composetypes.UpdateConfig{
						Delay:           5 * time.Second,
						FailureAction:   "rollback",
						Monitor:         10 * time.Second,
						MaxFailureRatio: 0.5,
					},
					RestartPolicy: &composetypes.RestartPolicy{
						Delay:       &duration,
						MaxAttempts: &attempts,
						Window:      &duration,
					},
				},
				Devices:       []string{"ignored"},
				DNSSearch:     []string{"ignored"},
				DNS:           []string{"ignored"},
				DomainName:    "ignored",
				EnvFile:       []string{"ignored"},
				Expose:        []string{"80"},
				ExternalLinks: []string{"ignored"},
				Image:         "dockerdemos/front",
				Links:         []string{"ignored"},
				Logging:       &composetypes.LoggingConfig{Driver: "syslog"},
				MacAddress:    "ignored",
				Networks:      map[string]*composetypes.ServiceNetworkConfig{"private": {}},
				NetworkMode:   "ignored",
				Restart:       "ignored",
				SecurityOpt:   []string{"ignored"},
				StopSignal:    "ignored",
				Ulimits:       map[string]*composetypes.UlimitsConfig{"nproc": {Hard: 65535}},
				User:          "ignored",
				Volumes: []composetypes.ServiceVolumeConfig{
					{
						Type: "bind",
						Bind: &composetypes.ServiceVolumeBind{Propagation: "ignored"},
					},
					{
						Type:   "volume",
						Volume: &composetypes.ServiceVolumeVolume{NoCopy: true},
					},
				},
			},
		},
		Networks: map[string]composetypes.NetworkConfig{
			"global": {},
		},
	}
	var buf bytes.Buffer
	warnUnsupportedFeatures(&buf, config)
	warnings := buf.String()
	golden.Assert(t, warnings, "warnings.golden")
}
