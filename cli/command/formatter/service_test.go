package formatter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/swarm"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/gotestyourself/gotestyourself/golden"
)

func TestServiceContextWrite(t *testing.T) {
	cases := []struct {
		context  Context
		expected string
	}{
		// Errors
		{
			Context{Format: "{{InvalidFunction}}"},
			`Template parsing error: template: :1: function "InvalidFunction" not defined
`,
		},
		{
			Context{Format: "{{nil}}"},
			`Template parsing error: template: :1:2: executing "" at <nil>: nil is not a command
`,
		},
		// Table format
		{
			Context{Format: NewServiceListFormat("table", false)},
			`ID                  NAME                MODE                REPLICAS            IMAGE               PORTS
id_baz              baz                 global              2/4                                     *:80->8080/tcp
id_bar              bar                 replicated          2/4                                     *:80->8080/tcp
`,
		},
		{
			Context{Format: NewServiceListFormat("table", true)},
			`id_baz
id_bar
`,
		},
		{
			Context{Format: NewServiceListFormat("table {{.Name}}", false)},
			`NAME
baz
bar
`,
		},
		{
			Context{Format: NewServiceListFormat("table {{.Name}}", true)},
			`NAME
baz
bar
`,
		},
		// Raw Format
		{
			Context{Format: NewServiceListFormat("raw", false)},
			string(golden.Get(t, "service-context-write-raw.golden")),
		},
		{
			Context{Format: NewServiceListFormat("raw", true)},
			`id: id_baz
id: id_bar
`,
		},
		// Custom Format
		{
			Context{Format: NewServiceListFormat("{{.Name}}", false)},
			`baz
bar
`,
		},
	}

	for _, testcase := range cases {
		services := []swarm.Service{
			{
				ID: "id_baz",
				Spec: swarm.ServiceSpec{
					Annotations: swarm.Annotations{Name: "baz"},
				},
				Endpoint: swarm.Endpoint{
					Ports: []swarm.PortConfig{
						{
							PublishMode:   "ingress",
							PublishedPort: 80,
							TargetPort:    8080,
							Protocol:      "tcp",
						},
					},
				},
			},
			{
				ID: "id_bar",
				Spec: swarm.ServiceSpec{
					Annotations: swarm.Annotations{Name: "bar"},
				},
				Endpoint: swarm.Endpoint{
					Ports: []swarm.PortConfig{
						{
							PublishMode:   "ingress",
							PublishedPort: 80,
							TargetPort:    8080,
							Protocol:      "tcp",
						},
					},
				},
			},
		}
		info := map[string]ServiceListInfo{
			"id_baz": {
				Mode:     "global",
				Replicas: "2/4",
			},
			"id_bar": {
				Mode:     "replicated",
				Replicas: "2/4",
			},
		}
		out := bytes.NewBufferString("")
		testcase.context.Output = out
		err := ServiceListWrite(testcase.context, services, info)
		if err != nil {
			assert.Error(t, err, testcase.expected)
		} else {
			assert.Check(t, is.Equal(testcase.expected, out.String()))
		}
	}
}

func TestServiceContextWriteJSON(t *testing.T) {
	services := []swarm.Service{
		{
			ID: "id_baz",
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{Name: "baz"},
			},
			Endpoint: swarm.Endpoint{
				Ports: []swarm.PortConfig{
					{
						PublishMode:   "ingress",
						PublishedPort: 80,
						TargetPort:    8080,
						Protocol:      "tcp",
					},
				},
			},
		},
		{
			ID: "id_bar",
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{Name: "bar"},
			},
			Endpoint: swarm.Endpoint{
				Ports: []swarm.PortConfig{
					{
						PublishMode:   "ingress",
						PublishedPort: 80,
						TargetPort:    8080,
						Protocol:      "tcp",
					},
				},
			},
		},
	}
	info := map[string]ServiceListInfo{
		"id_baz": {
			Mode:     "global",
			Replicas: "2/4",
		},
		"id_bar": {
			Mode:     "replicated",
			Replicas: "2/4",
		},
	}
	expectedJSONs := []map[string]interface{}{
		{"ID": "id_baz", "Name": "baz", "Mode": "global", "Replicas": "2/4", "Image": "", "Ports": "*:80->8080/tcp"},
		{"ID": "id_bar", "Name": "bar", "Mode": "replicated", "Replicas": "2/4", "Image": "", "Ports": "*:80->8080/tcp"},
	}

	out := bytes.NewBufferString("")
	err := ServiceListWrite(Context{Format: "{{json .}}", Output: out}, services, info)
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		msg := fmt.Sprintf("Output: line %d: %s", i, line)
		var m map[string]interface{}
		err := json.Unmarshal([]byte(line), &m)
		assert.NilError(t, err, msg)
		assert.Check(t, is.DeepEqual(expectedJSONs[i], m), msg)
	}
}
func TestServiceContextWriteJSONField(t *testing.T) {
	services := []swarm.Service{
		{ID: "id_baz", Spec: swarm.ServiceSpec{Annotations: swarm.Annotations{Name: "baz"}}},
		{ID: "id_bar", Spec: swarm.ServiceSpec{Annotations: swarm.Annotations{Name: "bar"}}},
	}
	info := map[string]ServiceListInfo{
		"id_baz": {
			Mode:     "global",
			Replicas: "2/4",
		},
		"id_bar": {
			Mode:     "replicated",
			Replicas: "2/4",
		},
	}
	out := bytes.NewBufferString("")
	err := ServiceListWrite(Context{Format: "{{json .Name}}", Output: out}, services, info)
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		msg := fmt.Sprintf("Output: line %d: %s", i, line)
		var s string
		err := json.Unmarshal([]byte(line), &s)
		assert.NilError(t, err, msg)
		assert.Check(t, is.Equal(services[i].Spec.Name, s), msg)
	}
}

func TestServiceContext_Ports(t *testing.T) {
	c := serviceContext{
		service: swarm.Service{
			Endpoint: swarm.Endpoint{
				Ports: []swarm.PortConfig{
					{
						Protocol:      "tcp",
						TargetPort:    80,
						PublishedPort: 81,
						PublishMode:   "ingress",
					},
					{
						Protocol:      "tcp",
						TargetPort:    80,
						PublishedPort: 80,
						PublishMode:   "ingress",
					},
					{
						Protocol:      "tcp",
						TargetPort:    95,
						PublishedPort: 95,
						PublishMode:   "ingress",
					},
					{
						Protocol:      "tcp",
						TargetPort:    90,
						PublishedPort: 90,
						PublishMode:   "ingress",
					},
					{
						Protocol:      "tcp",
						TargetPort:    91,
						PublishedPort: 91,
						PublishMode:   "ingress",
					},
					{
						Protocol:      "tcp",
						TargetPort:    92,
						PublishedPort: 92,
						PublishMode:   "ingress",
					},
					{
						Protocol:      "tcp",
						TargetPort:    93,
						PublishedPort: 93,
						PublishMode:   "ingress",
					},
					{
						Protocol:      "tcp",
						TargetPort:    94,
						PublishedPort: 94,
						PublishMode:   "ingress",
					},
					{
						Protocol:      "udp",
						TargetPort:    95,
						PublishedPort: 95,
						PublishMode:   "ingress",
					},
					{
						Protocol:      "udp",
						TargetPort:    90,
						PublishedPort: 90,
						PublishMode:   "ingress",
					},
					{
						Protocol:      "udp",
						TargetPort:    96,
						PublishedPort: 96,
						PublishMode:   "ingress",
					},
					{
						Protocol:      "udp",
						TargetPort:    91,
						PublishedPort: 91,
						PublishMode:   "ingress",
					},
					{
						Protocol:      "udp",
						TargetPort:    92,
						PublishedPort: 92,
						PublishMode:   "ingress",
					},
					{
						Protocol:      "udp",
						TargetPort:    93,
						PublishedPort: 93,
						PublishMode:   "ingress",
					},
					{
						Protocol:      "udp",
						TargetPort:    94,
						PublishedPort: 94,
						PublishMode:   "ingress",
					},
					{
						Protocol:      "tcp",
						TargetPort:    60,
						PublishedPort: 60,
						PublishMode:   "ingress",
					},
					{
						Protocol:      "tcp",
						TargetPort:    61,
						PublishedPort: 61,
						PublishMode:   "ingress",
					},
					{
						Protocol:      "tcp",
						TargetPort:    61,
						PublishedPort: 62,
						PublishMode:   "ingress",
					},
					{
						Protocol:      "sctp",
						TargetPort:    97,
						PublishedPort: 97,
						PublishMode:   "ingress",
					},
					{
						Protocol:      "sctp",
						TargetPort:    98,
						PublishedPort: 98,
						PublishMode:   "ingress",
					},
				},
			},
		},
	}

	assert.Check(t, is.Equal("*:97-98->97-98/sctp, *:60-61->60-61/tcp, *:62->61/tcp, *:80-81->80/tcp, *:90-95->90-95/tcp, *:90-96->90-96/udp", c.Ports()))
}
