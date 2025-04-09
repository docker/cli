package system

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types/events"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

func TestEventsFormat(t *testing.T) {
	var evts []events.Message //nolint:prealloc
	for i, action := range []events.Action{events.ActionCreate, events.ActionStart, events.ActionAttach, events.ActionDie} {
		evts = append(evts, events.Message{
			Status: string(action),
			ID:     "abc123",
			From:   "ubuntu:latest",
			Type:   events.ContainerEventType,
			Action: action,
			Actor: events.Actor{
				ID:         "abc123",
				Attributes: map[string]string{"image": "ubuntu:latest"},
			},
			Scope:    "local",
			Time:     int64(time.Second) * int64(i+1),
			TimeNano: int64(time.Second) * int64(i+1),
		})
	}
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "default",
			args: []string{},
		},
		{
			name: "json",
			args: []string{"--format", "json"},
		},
		{
			name: "json template",
			args: []string{"--format", "{{ json . }}"},
		},
		{
			name: "json action",
			args: []string{"--format", "{{ json .Action }}"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set to UTC timezone as timestamps in output are
			// printed in the current timezone
			t.Setenv("TZ", "UTC")
			cli := test.NewFakeCli(&fakeClient{eventsFn: func(context.Context, events.ListOptions) (<-chan events.Message, <-chan error) {
				messages := make(chan events.Message)
				errs := make(chan error, 1)
				go func() {
					for _, msg := range evts {
						messages <- msg
					}
					errs <- io.EOF
				}()
				return messages, errs
			}})
			cmd := NewEventsCommand(cli)
			cmd.SetArgs(tc.args)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			assert.Check(t, cmd.Execute())
			out := cli.OutBuffer().String()
			assert.Check(t, golden.String(out, fmt.Sprintf("docker-events-%s.golden", strings.ReplaceAll(tc.name, " ", "-"))))
			cli.OutBuffer().Reset()
		})
	}
}
