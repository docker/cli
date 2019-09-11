package container

import (
	"bytes"
	"testing"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/docker/pkg/stringid"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func TestContainerStatsContext(t *testing.T) {
	containerID := stringid.GenerateRandomID()

	var ctx statsContext
	tt := []struct {
		stats     StatsEntry
		osType    string
		expValue  string
		expHeader string
		call      func() string
	}{
		{StatsEntry{Container: containerID}, "", containerID, containerHeader, ctx.Container},
		{StatsEntry{CPUPercentage: 5.5}, "", "5.50%", cpuPercHeader, ctx.CPUPerc},
		{StatsEntry{CPUPercentage: 5.5, IsInvalid: true}, "", "--", cpuPercHeader, ctx.CPUPerc},
		{StatsEntry{NetworkRx: 0.31, NetworkTx: 12.3}, "", "0.31B / 12.3B", netIOHeader, ctx.NetIO},
		{StatsEntry{NetworkRx: 0.31, NetworkTx: 12.3, IsInvalid: true}, "", "--", netIOHeader, ctx.NetIO},
		{StatsEntry{BlockRead: 0.1, BlockWrite: 2.3}, "", "0.1B / 2.3B", blockIOHeader, ctx.BlockIO},
		{StatsEntry{BlockRead: 0.1, BlockWrite: 2.3, IsInvalid: true}, "", "--", blockIOHeader, ctx.BlockIO},
		{StatsEntry{MemoryPercentage: 10.2}, "", "10.20%", memPercHeader, ctx.MemPerc},
		{StatsEntry{MemoryPercentage: 10.2, IsInvalid: true}, "", "--", memPercHeader, ctx.MemPerc},
		{StatsEntry{MemoryPercentage: 10.2}, "windows", "--", memPercHeader, ctx.MemPerc},
		{StatsEntry{Memory: 24, MemoryLimit: 30}, "", "24B / 30B", memUseHeader, ctx.MemUsage},
		{StatsEntry{Memory: 24, MemoryLimit: 30, IsInvalid: true}, "", "-- / --", memUseHeader, ctx.MemUsage},
		{StatsEntry{Memory: 24, MemoryLimit: 30}, "windows", "24B", winMemUseHeader, ctx.MemUsage},
		{StatsEntry{PidsCurrent: 10}, "", "10", pidsHeader, ctx.PIDs},
		{StatsEntry{PidsCurrent: 10, IsInvalid: true}, "", "--", pidsHeader, ctx.PIDs},
		{StatsEntry{PidsCurrent: 10}, "windows", "--", pidsHeader, ctx.PIDs},
	}

	for _, te := range tt {
		ctx = statsContext{s: te.stats, os: te.osType}
		if v := te.call(); v != te.expValue {
			t.Fatalf("Expected %q, got %q", te.expValue, v)
		}
	}
}

func TestContainerStatsContextWrite(t *testing.T) {
	tt := []struct {
		context  formatter.Context
		expected string
	}{
		{
			formatter.Context{Format: "{{InvalidFunction}}"},
			`Template parsing error: template: :1: function "InvalidFunction" not defined
`,
		},
		{
			formatter.Context{Format: "{{nil}}"},
			`Template parsing error: template: :1:2: executing "" at <nil>: nil is not a command
`,
		},
		{
			formatter.Context{Format: "table {{.MemUsage}}"},
			`MEM USAGE / LIMIT
20B / 20B
-- / --
`,
		},
		{
			formatter.Context{Format: "{{.Container}}  {{.ID}}  {{.Name}}"},
			`container1  abcdef  foo
container2    --
`,
		},
		{
			formatter.Context{Format: "{{.Container}}  {{.CPUPerc}}"},
			`container1  20.00%
container2  --
`,
		},
	}

	for _, te := range tt {
		stats := []StatsEntry{
			{
				Container:        "container1",
				ID:               "abcdef",
				Name:             "/foo",
				CPUPercentage:    20,
				Memory:           20,
				MemoryLimit:      20,
				MemoryPercentage: 20,
				NetworkRx:        20,
				NetworkTx:        20,
				BlockRead:        20,
				BlockWrite:       20,
				PidsCurrent:      2,
				IsInvalid:        false,
			},
			{
				Container:        "container2",
				CPUPercentage:    30,
				Memory:           30,
				MemoryLimit:      30,
				MemoryPercentage: 30,
				NetworkRx:        30,
				NetworkTx:        30,
				BlockRead:        30,
				BlockWrite:       30,
				PidsCurrent:      3,
				IsInvalid:        true,
			},
		}
		var out bytes.Buffer
		te.context.Output = &out
		err := statsFormatWrite(te.context, stats, "linux", false)
		if err != nil {
			assert.Error(t, err, te.expected)
		} else {
			assert.Check(t, is.Equal(te.expected, out.String()))
		}
	}
}

func TestContainerStatsContextWriteWindows(t *testing.T) {
	tt := []struct {
		context  formatter.Context
		expected string
	}{
		{
			formatter.Context{Format: "table {{.MemUsage}}"},
			`PRIV WORKING SET
20B
-- / --
`,
		},
		{
			formatter.Context{Format: "{{.Container}}  {{.CPUPerc}}"},
			`container1  20.00%
container2  --
`,
		},
		{
			formatter.Context{Format: "{{.Container}}  {{.MemPerc}}  {{.PIDs}}"},
			`container1  --  --
container2  --  --
`,
		},
	}

	for _, te := range tt {
		stats := []StatsEntry{
			{
				Container:        "container1",
				CPUPercentage:    20,
				Memory:           20,
				MemoryLimit:      20,
				MemoryPercentage: 20,
				NetworkRx:        20,
				NetworkTx:        20,
				BlockRead:        20,
				BlockWrite:       20,
				PidsCurrent:      2,
				IsInvalid:        false,
			},
			{
				Container:        "container2",
				CPUPercentage:    30,
				Memory:           30,
				MemoryLimit:      30,
				MemoryPercentage: 30,
				NetworkRx:        30,
				NetworkTx:        30,
				BlockRead:        30,
				BlockWrite:       30,
				PidsCurrent:      3,
				IsInvalid:        true,
			},
		}
		var out bytes.Buffer
		te.context.Output = &out
		err := statsFormatWrite(te.context, stats, "windows", false)
		if err != nil {
			assert.Error(t, err, te.expected)
		} else {
			assert.Check(t, is.Equal(te.expected, out.String()))
		}
	}
}

func TestContainerStatsContextWriteWithNoStats(t *testing.T) {
	var out bytes.Buffer

	contexts := []struct {
		context  formatter.Context
		expected string
	}{
		{
			formatter.Context{
				Format: "{{.Container}}",
				Output: &out,
			},
			"",
		},
		{
			formatter.Context{
				Format: "table {{.Container}}",
				Output: &out,
			},
			"CONTAINER\n",
		},
		{
			formatter.Context{
				Format: "table {{.Container}}\t{{.CPUPerc}}",
				Output: &out,
			},
			"CONTAINER           CPU %\n",
		},
	}

	for _, context := range contexts {
		statsFormatWrite(context.context, []StatsEntry{}, "linux", false)
		assert.Check(t, is.Equal(context.expected, out.String()))
		// Clean buffer
		out.Reset()
	}
}

func TestContainerStatsContextWriteWithNoStatsWindows(t *testing.T) {
	var out bytes.Buffer

	contexts := []struct {
		context  formatter.Context
		expected string
	}{
		{
			formatter.Context{
				Format: "{{.Container}}",
				Output: &out,
			},
			"",
		},
		{
			formatter.Context{
				Format: "table {{.Container}}\t{{.MemUsage}}",
				Output: &out,
			},
			"CONTAINER           PRIV WORKING SET\n",
		},
		{
			formatter.Context{
				Format: "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}",
				Output: &out,
			},
			"CONTAINER           CPU %               PRIV WORKING SET\n",
		},
	}

	for _, context := range contexts {
		statsFormatWrite(context.context, []StatsEntry{}, "windows", false)
		assert.Check(t, is.Equal(context.expected, out.String()))
		// Clean buffer
		out.Reset()
	}
}

func TestContainerStatsContextWriteTrunc(t *testing.T) {
	var out bytes.Buffer

	contexts := []struct {
		context  formatter.Context
		trunc    bool
		expected string
	}{
		{
			formatter.Context{
				Format: "{{.ID}}",
				Output: &out,
			},
			false,
			"b95a83497c9161c9b444e3d70e1a9dfba0c1840d41720e146a95a08ebf938afc\n",
		},
		{
			formatter.Context{
				Format: "{{.ID}}",
				Output: &out,
			},
			true,
			"b95a83497c91\n",
		},
	}

	for _, context := range contexts {
		statsFormatWrite(context.context, []StatsEntry{{ID: "b95a83497c9161c9b444e3d70e1a9dfba0c1840d41720e146a95a08ebf938afc"}}, "linux", context.trunc)
		assert.Check(t, is.Equal(context.expected, out.String()))
		// Clean buffer
		out.Reset()
	}
}
