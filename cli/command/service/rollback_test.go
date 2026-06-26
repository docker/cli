package service

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestRollback(t *testing.T) {
	testCases := []struct {
		name                 string
		args                 []string
		serviceUpdateFunc    func(ctx context.Context, serviceID string, options client.ServiceUpdateOptions) (client.ServiceUpdateResult, error)
		expectedDockerCliErr string
	}{
		{
			name: "rollback-service",
			args: []string{"service-id"},
		},
		{
			name: "rollback-service-with-warnings",
			args: []string{"service-id"},
			serviceUpdateFunc: func(ctx context.Context, serviceID string, options client.ServiceUpdateOptions) (client.ServiceUpdateResult, error) {
				return client.ServiceUpdateResult{
					Warnings: []string{
						"- warning 1",
						"- warning 2",
					},
				}, nil
			},
			expectedDockerCliErr: "- warning 1\n- warning 2",
		},
	}

	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{
			serviceUpdateFunc: tc.serviceUpdateFunc,
		})
		cmd := newRollbackCommand(cli)
		cmd.SetArgs(tc.args)
		assert.NilError(t, cmd.Flags().Set("quiet", "true"))
		assert.NilError(t, cmd.Flags().Set("detach", "true"))
		cmd.SetOut(io.Discard)
		assert.NilError(t, cmd.Execute())
		assert.Check(t, is.Equal(strings.TrimSpace(cli.ErrBuffer().String()), tc.expectedDockerCliErr))
	}
}

func TestRollbackWithErrors(t *testing.T) {
	testCases := []struct {
		name               string
		args               []string
		serviceInspectFunc func(ctx context.Context, serviceID string, options client.ServiceInspectOptions) (client.ServiceInspectResult, error)
		serviceUpdateFunc  func(ctx context.Context, serviceID string, options client.ServiceUpdateOptions) (client.ServiceUpdateResult, error)
		expectedError      string
	}{
		{
			name:          "not-enough-args",
			expectedError: "requires 1 argument",
		},
		{
			name:          "too-many-args",
			args:          []string{"service-id-1", "service-id-2"},
			expectedError: "requires 1 argument",
		},
		{
			name: "service-does-not-exists",
			args: []string{"service-id"},
			serviceInspectFunc: func(ctx context.Context, serviceID string, options client.ServiceInspectOptions) (client.ServiceInspectResult, error) {
				return client.ServiceInspectResult{}, fmt.Errorf("no such services: %s", serviceID)
			},
			expectedError: "no such services: service-id",
		},
		{
			name: "service-update-failed",
			args: []string{"service-id"},
			serviceUpdateFunc: func(ctx context.Context, serviceID string, options client.ServiceUpdateOptions) (client.ServiceUpdateResult, error) {
				return client.ServiceUpdateResult{}, fmt.Errorf("no such services: %s", serviceID)
			},
			expectedError: "no such services: service-id",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newRollbackCommand(
				test.NewFakeCli(&fakeClient{
					serviceInspectFunc: tc.serviceInspectFunc,
					serviceUpdateFunc:  tc.serviceUpdateFunc,
				}))
			cmd.SetArgs(tc.args)
			assert.NilError(t, cmd.Flags().Set("quiet", "true"))
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
		})
	}
}
