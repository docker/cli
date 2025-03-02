// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.22

package service

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/golden"
)

func formatServiceInspect(t *testing.T, format formatter.Format, now time.Time) string {
	t.Helper()
	b := new(bytes.Buffer)

	endpointSpec := &swarm.EndpointSpec{
		Mode: "vip",
		Ports: []swarm.PortConfig{
			{
				Protocol:   swarm.PortConfigProtocolTCP,
				TargetPort: 5000,
			},
		},
	}

	two := uint64(2)

	s := swarm.Service{
		ID: "de179gar9d0o7ltdybungplod",
		Meta: swarm.Meta{
			Version:   swarm.Version{Index: 315},
			CreatedAt: now,
			UpdatedAt: now,
		},
		Spec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name:   "my_service",
				Labels: map[string]string{"com.label": "foo"},
			},
			TaskTemplate: swarm.TaskSpec{
				LogDriver: &swarm.Driver{
					Name: "driver",
					Options: map[string]string{
						"max-file": "5",
					},
				},
				ContainerSpec: &swarm.ContainerSpec{
					Image: "foo/bar@sha256:this_is_a_test",
					Configs: []*swarm.ConfigReference{
						{
							ConfigID:   "mtc3i44r1awdoziy2iceg73z8",
							ConfigName: "configtest.conf",
							File: &swarm.ConfigReferenceFileTarget{
								Name: "/configtest.conf",
							},
						},
					},
					Secrets: []*swarm.SecretReference{
						{
							SecretID:   "3hv39ehbbb4hdozo7spod9ftn",
							SecretName: "secrettest.conf",
							File: &swarm.SecretReferenceFileTarget{
								Name: "/secrettest.conf",
							},
						},
					},

					Healthcheck: &container.HealthConfig{
						Test:        []string{"CMD-SHELL", "curl"},
						Interval:    4,
						Retries:     3,
						StartPeriod: 2,
						Timeout:     1,
					},
				},
				Resources: &swarm.ResourceRequirements{
					Limits: &swarm.Limit{
						NanoCPUs:    100000000000,
						MemoryBytes: 10490000,
						Pids:        20,
					},
				},
				Networks: []swarm.NetworkAttachmentConfig{
					{
						Target:  "5vpyomhb6ievnk0i0o60gcnei",
						Aliases: []string{"web"},
					},
				},
			},
			Mode: swarm.ServiceMode{
				Replicated: &swarm.ReplicatedService{
					Replicas: &two,
				},
			},
			EndpointSpec: endpointSpec,
		},
		Endpoint: swarm.Endpoint{
			Spec: *endpointSpec,
			Ports: []swarm.PortConfig{
				{
					Protocol:      swarm.PortConfigProtocolTCP,
					TargetPort:    5000,
					PublishedPort: 30000,
				},
			},
			VirtualIPs: []swarm.EndpointVirtualIP{
				{
					NetworkID: "6o4107cj2jx9tihgb0jyts6pj",
					Addr:      "10.255.0.4/16",
				},
			},
		},
		UpdateStatus: &swarm.UpdateStatus{
			StartedAt:   &now,
			CompletedAt: &now,
		},
	}

	ctx := formatter.Context{
		Output: b,
		Format: format,
	}

	err := InspectFormatWrite(ctx, []string{"de179gar9d0o7ltdybungplod"},
		func(ref string) (any, []byte, error) {
			return s, nil, nil
		},
		func(ref string) (any, []byte, error) {
			return network.Inspect{
				ID:   "5vpyomhb6ievnk0i0o60gcnei",
				Name: "mynetwork",
			}, nil, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func TestPrettyPrint(t *testing.T) {
	s := formatServiceInspect(t, NewFormat("pretty"), time.Now())
	golden.Assert(t, s, "service-inspect-pretty.golden")
}

func TestPrettyPrintWithNoUpdateConfig(t *testing.T) {
	s := formatServiceInspect(t, NewFormat("pretty"), time.Now())
	if strings.Contains(s, "UpdateStatus") {
		t.Fatal("Pretty print failed before parsing UpdateStatus")
	}
	if !strings.Contains(s, "mynetwork") {
		t.Fatal("network name not found in inspect output")
	}
}

func TestJSONFormatWithNoUpdateConfig(t *testing.T) {
	now := time.Now()
	// s1: [{"ID":..}]
	// s2: {"ID":..}
	s1 := formatServiceInspect(t, NewFormat(""), now)
	s2 := formatServiceInspect(t, NewFormat("{{json .}}"), now)
	var m1Wrap []map[string]any
	if err := json.Unmarshal([]byte(s1), &m1Wrap); err != nil {
		t.Fatal(err)
	}
	if len(m1Wrap) != 1 {
		t.Fatalf("strange s1=%s", s1)
	}
	m1 := m1Wrap[0]
	var m2 map[string]any
	if err := json.Unmarshal([]byte(s2), &m2); err != nil {
		t.Fatal(err)
	}
	assert.Check(t, is.DeepEqual(m1, m2))
}

func TestPrettyPrintWithConfigsAndSecrets(t *testing.T) {
	s := formatServiceInspect(t, NewFormat("pretty"), time.Now())
	assert.Check(t, is.Contains(s, "Log Driver:"), "Pretty print missing Log Driver")
	assert.Check(t, is.Contains(s, "Configs:"), "Pretty print missing configs")
	assert.Check(t, is.Contains(s, "Secrets:"), "Pretty print missing secrets")
	assert.Check(t, is.Contains(s, "Healthcheck:"), "Pretty print missing healthcheck")
}
