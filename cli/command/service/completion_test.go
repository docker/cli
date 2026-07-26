package service

import (
	"context"
	"errors"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/builders"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
)

func TestCompleteServicePsFilters(t *testing.T) {
	tests := []struct {
		doc        string
		client     *fakeClient
		toComplete string
		expected   []string
		directive  cobra.ShellCompDirective
	}{
		{
			doc:        "no input offers the filter keys",
			toComplete: "",
			expected:   []string{"desired-state=", "id=", "name=", "node="},
			directive:  cobra.ShellCompDirectiveNoSpace,
		},
		{
			doc:        "desired-state values",
			toComplete: "desired-state=",
			expected:   []string{"desired-state=running", "desired-state=shutdown", "desired-state=accepted"},
			directive:  cobra.ShellCompDirectiveNoFileComp,
		},
		{
			doc: "node values",
			client: &fakeClient{
				nodeListFunc: func(_ context.Context, _ client.NodeListOptions) (client.NodeListResult, error) {
					return client.NodeListResult{
						Items: []swarm.Node{
							*builders.Node(builders.Hostname("n1")),
							*builders.Node(builders.Hostname("n2")),
						},
					}, nil
				},
			},
			toComplete: "node=",
			expected:   []string{"node=n1", "node=n2"},
			directive:  cobra.ShellCompDirectiveNoFileComp,
		},
		{
			doc: "node values on API error",
			client: &fakeClient{
				nodeListFunc: func(_ context.Context, _ client.NodeListOptions) (client.NodeListResult, error) {
					return client.NodeListResult{}, errors.New("API error")
				},
			},
			toComplete: "node=",
			expected:   []string{},
			directive:  cobra.ShellCompDirectiveNoFileComp,
		},
		{
			doc:        "id offers no values",
			toComplete: "id=",
			expected:   nil,
			directive:  cobra.ShellCompDirectiveNoFileComp,
		},
		{
			doc:        "unknown key falls back to the filter keys",
			toComplete: "bogus=",
			expected:   []string{"desired-state=", "id=", "name=", "node="},
			directive:  cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp,
		},
	}

	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			cli := test.NewFakeCli(tc.client)
			completions, directive := completeServicePsFilters(cli)(newPsCommand(cli), nil, tc.toComplete)
			assert.DeepEqual(t, completions, tc.expected)
			assert.Equal(t, directive, tc.directive)
		})
	}
}

func TestCompleteServiceListFilters(t *testing.T) {
	tests := []struct {
		doc        string
		client     *fakeClient
		toComplete string
		expected   []string
		directive  cobra.ShellCompDirective
	}{
		{
			doc:        "no input offers the filter keys",
			toComplete: "",
			expected:   []string{"id=", "label=", "mode=", "name="},
			directive:  cobra.ShellCompDirectiveNoSpace,
		},
		{
			doc:        "mode values",
			toComplete: "mode=",
			expected:   []string{"mode=replicated", "mode=global", "mode=replicated-job", "mode=global-job"},
			directive:  cobra.ShellCompDirectiveNoFileComp,
		},
		{
			doc: "name values",
			client: &fakeClient{
				serviceListFunc: func(_ context.Context, _ client.ServiceListOptions) (client.ServiceListResult, error) {
					return client.ServiceListResult{
						Items: []swarm.Service{
							*builders.Service(builders.ServiceName("s1")),
							*builders.Service(builders.ServiceName("s2")),
						},
					}, nil
				},
			},
			toComplete: "name=",
			expected:   []string{"name=s1", "name=s2"},
			directive:  cobra.ShellCompDirectiveNoFileComp,
		},
		{
			doc: "name values on API error",
			client: &fakeClient{
				serviceListFunc: func(_ context.Context, _ client.ServiceListOptions) (client.ServiceListResult, error) {
					return client.ServiceListResult{}, errors.New("API error")
				},
			},
			toComplete: "name=",
			expected:   []string{},
			directive:  cobra.ShellCompDirectiveNoFileComp,
		},
		{
			doc:        "label offers no values",
			toComplete: "label=",
			expected:   nil,
			directive:  cobra.ShellCompDirectiveNoFileComp,
		},
		{
			doc:        "unknown key falls back to the filter keys",
			toComplete: "bogus=",
			expected:   []string{"id=", "label=", "mode=", "name="},
			directive:  cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp,
		},
	}

	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			cli := test.NewFakeCli(tc.client)
			completions, directive := completeServiceListFilters(cli)(newListCommand(cli), nil, tc.toComplete)
			assert.DeepEqual(t, completions, tc.expected)
			assert.Equal(t, directive, tc.directive)
		})
	}
}
