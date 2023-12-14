// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.19

package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/docker/api/types/swarm"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/golden"
)

func TestServiceContextWrite(t *testing.T) {
	var (
		// we need a pair of variables for setting the job parameters, because
		// those parameters take pointers to uint64, which we can't make as a
		// literal
		varThree uint64 = 3
		varTen   uint64 = 10
	)
	cases := []struct {
		context  formatter.Context
		expected string
	}{
		// Errors
		{
			formatter.Context{Format: "{{InvalidFunction}}"},
			`template parsing error: template: :1: function "InvalidFunction" not defined`,
		},
		{
			formatter.Context{Format: "{{nil}}"},
			`template parsing error: template: :1:2: executing "" at <nil>: nil is not a command`,
		},
		// Table format
		{
			formatter.Context{Format: NewListFormat("table", false)},
			`ID         NAME      MODE             REPLICAS               IMAGE     PORTS
02_bar     bar       replicated       2/4                              *:80->8090/udp
01_baz     baz       global           1/3                              *:80->8080/tcp
04_qux2    qux2      replicated       3/3 (max 2 per node)             
03_qux10   qux10     replicated       2/3 (max 1 per node)             
05_job1    zarp1     replicated job   2/3 (5/10 completed)             
06_job2    zarp2     global job       1/1 (3/4 completed)              
`,
		},
		{
			formatter.Context{Format: NewListFormat("table", true)},
			`02_bar
01_baz
04_qux2
03_qux10
05_job1
06_job2
`,
		},
		{
			formatter.Context{Format: NewListFormat("table {{.Name}}\t{{.Mode}}", false)},
			`NAME      MODE
bar       replicated
baz       global
qux2      replicated
qux10     replicated
zarp1     replicated job
zarp2     global job
`,
		},
		{
			formatter.Context{Format: NewListFormat("table {{.Name}}", true)},
			`NAME
bar
baz
qux2
qux10
zarp1
zarp2
`,
		},
		// Raw Format
		{
			formatter.Context{Format: NewListFormat("raw", false)},
			string(golden.Get(t, "service-context-write-raw.golden")),
		},
		{
			formatter.Context{Format: NewListFormat("raw", true)},
			`id: 02_bar
id: 01_baz
id: 04_qux2
id: 03_qux10
id: 05_job1
id: 06_job2
`,
		},
		// Custom Format
		{
			formatter.Context{Format: NewListFormat("{{.Name}}", false)},
			`bar
baz
qux2
qux10
zarp1
zarp2
`,
		},
	}

	services := []swarm.Service{
		{
			ID: "01_baz",
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{Name: "baz"},
				Mode: swarm.ServiceMode{
					Global: &swarm.GlobalService{},
				},
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
			ServiceStatus: &swarm.ServiceStatus{
				RunningTasks: 1,
				DesiredTasks: 3,
			},
		},
		{
			ID: "02_bar",
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{Name: "bar"},
				Mode: swarm.ServiceMode{
					Replicated: &swarm.ReplicatedService{},
				},
			},
			Endpoint: swarm.Endpoint{
				Ports: []swarm.PortConfig{
					{
						PublishMode:   "ingress",
						PublishedPort: 80,
						TargetPort:    8090,
						Protocol:      "udp",
					},
				},
			},
			ServiceStatus: &swarm.ServiceStatus{
				RunningTasks: 2,
				DesiredTasks: 4,
			},
		},
		{
			ID: "03_qux10",
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{Name: "qux10"},
				Mode: swarm.ServiceMode{
					Replicated: &swarm.ReplicatedService{},
				},
				TaskTemplate: swarm.TaskSpec{
					Placement: &swarm.Placement{MaxReplicas: 1},
				},
			},
			ServiceStatus: &swarm.ServiceStatus{
				RunningTasks: 2,
				DesiredTasks: 3,
			},
		},
		{
			ID: "04_qux2",
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{Name: "qux2"},
				Mode: swarm.ServiceMode{
					Replicated: &swarm.ReplicatedService{},
				},
				TaskTemplate: swarm.TaskSpec{
					Placement: &swarm.Placement{MaxReplicas: 2},
				},
			},
			ServiceStatus: &swarm.ServiceStatus{
				RunningTasks: 3,
				DesiredTasks: 3,
			},
		},
		{
			ID: "05_job1",
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{Name: "zarp1"},
				Mode: swarm.ServiceMode{
					ReplicatedJob: &swarm.ReplicatedJob{
						MaxConcurrent:    &varThree,
						TotalCompletions: &varTen,
					},
				},
			},
			ServiceStatus: &swarm.ServiceStatus{
				RunningTasks:   2,
				DesiredTasks:   3,
				CompletedTasks: 5,
			},
		},
		{
			ID: "06_job2",
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{Name: "zarp2"},
				Mode: swarm.ServiceMode{
					GlobalJob: &swarm.GlobalJob{},
				},
			},
			ServiceStatus: &swarm.ServiceStatus{
				RunningTasks:   1,
				DesiredTasks:   1,
				CompletedTasks: 3,
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(string(tc.context.Format), func(t *testing.T) {
			var out bytes.Buffer
			tc.context.Output = &out

			if err := ListFormatWrite(tc.context, services); err != nil {
				assert.Error(t, err, tc.expected)
			} else {
				assert.Equal(t, out.String(), tc.expected)
			}
		})
	}
}

func TestServiceContextWriteJSON(t *testing.T) {
	services := []swarm.Service{
		{
			ID: "01_baz",
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{Name: "baz"},
				Mode: swarm.ServiceMode{
					Global: &swarm.GlobalService{},
				},
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
			ServiceStatus: &swarm.ServiceStatus{
				RunningTasks: 1,
				DesiredTasks: 3,
			},
		},
		{
			ID: "02_bar",
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{Name: "bar"},
				Mode: swarm.ServiceMode{
					Replicated: &swarm.ReplicatedService{},
				},
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
			ServiceStatus: &swarm.ServiceStatus{
				RunningTasks: 2,
				DesiredTasks: 4,
			},
		},
	}
	expectedJSONs := []map[string]interface{}{
		{"ID": "02_bar", "Name": "bar", "Mode": "replicated", "Replicas": "2/4", "Image": "", "Ports": "*:80->8080/tcp"},
		{"ID": "01_baz", "Name": "baz", "Mode": "global", "Replicas": "1/3", "Image": "", "Ports": "*:80->8080/tcp"},
	}

	out := bytes.NewBufferString("")
	err := ListFormatWrite(formatter.Context{Format: "{{json .}}", Output: out}, services)
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
		{
			ID: "01_baz",
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{Name: "baz"},
				Mode: swarm.ServiceMode{
					Global: &swarm.GlobalService{},
				},
			},
			ServiceStatus: &swarm.ServiceStatus{
				RunningTasks: 2,
				DesiredTasks: 4,
			},
		},
		{
			ID: "24_bar",
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{Name: "bar"},
				Mode: swarm.ServiceMode{
					Replicated: &swarm.ReplicatedService{},
				},
			},
			ServiceStatus: &swarm.ServiceStatus{
				RunningTasks: 2,
				DesiredTasks: 4,
			},
		},
	}
	out := bytes.NewBufferString("")
	err := ListFormatWrite(formatter.Context{Format: "{{json .Name}}", Output: out}, services)
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
