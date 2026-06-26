package swarm

import (
	"context"

	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/client"
)

type fakeClient struct {
	client.Client
	infoFunc              func() (system.Info, error)
	swarmInitFunc         func(client.SwarmInitOptions) (client.SwarmInitResult, error)
	swarmInspectFunc      func() (client.SwarmInspectResult, error)
	nodeInspectFunc       func() (client.NodeInspectResult, error)
	swarmGetUnlockKeyFunc func() (client.SwarmGetUnlockKeyResult, error)
	swarmJoinFunc         func() (client.SwarmJoinResult, error)
	swarmLeaveFunc        func() (client.SwarmLeaveResult, error)
	swarmUpdateFunc       func(client.SwarmUpdateOptions) (client.SwarmUpdateResult, error)
	swarmUnlockFunc       func(client.SwarmUnlockOptions) (client.SwarmUnlockResult, error)
}

func (cli *fakeClient) Info(context.Context, client.InfoOptions) (client.SystemInfoResult, error) {
	if cli.infoFunc != nil {
		inf, err := cli.infoFunc()
		return client.SystemInfoResult{
			Info: inf,
		}, err
	}
	return client.SystemInfoResult{}, nil
}

func (cli *fakeClient) NodeInspect(context.Context, string, client.NodeInspectOptions) (client.NodeInspectResult, error) {
	if cli.nodeInspectFunc != nil {
		return cli.nodeInspectFunc()
	}
	return client.NodeInspectResult{}, nil
}

func (cli *fakeClient) SwarmInit(_ context.Context, options client.SwarmInitOptions) (client.SwarmInitResult, error) {
	if cli.swarmInitFunc != nil {
		return cli.swarmInitFunc(options)
	}
	return client.SwarmInitResult{}, nil
}

func (cli *fakeClient) SwarmInspect(context.Context, client.SwarmInspectOptions) (client.SwarmInspectResult, error) {
	if cli.swarmInspectFunc != nil {
		return cli.swarmInspectFunc()
	}
	return client.SwarmInspectResult{}, nil
}

func (cli *fakeClient) SwarmGetUnlockKey(ctx context.Context) (client.SwarmGetUnlockKeyResult, error) {
	if cli.swarmGetUnlockKeyFunc != nil {
		return cli.swarmGetUnlockKeyFunc()
	}
	return client.SwarmGetUnlockKeyResult{}, nil
}

func (cli *fakeClient) SwarmJoin(context.Context, client.SwarmJoinOptions) (client.SwarmJoinResult, error) {
	if cli.swarmJoinFunc != nil {
		return cli.swarmJoinFunc()
	}
	return client.SwarmJoinResult{}, nil
}

func (cli *fakeClient) SwarmLeave(context.Context, client.SwarmLeaveOptions) (client.SwarmLeaveResult, error) {
	if cli.swarmLeaveFunc != nil {
		return cli.swarmLeaveFunc()
	}
	return client.SwarmLeaveResult{}, nil
}

func (cli *fakeClient) SwarmUpdate(_ context.Context, options client.SwarmUpdateOptions) (client.SwarmUpdateResult, error) {
	if cli.swarmUpdateFunc != nil {
		return cli.swarmUpdateFunc(options)
	}
	return client.SwarmUpdateResult{}, nil
}

func (cli *fakeClient) SwarmUnlock(_ context.Context, options client.SwarmUnlockOptions) (client.SwarmUnlockResult, error) {
	if cli.swarmUnlockFunc != nil {
		return cli.swarmUnlockFunc(options)
	}
	return client.SwarmUnlockResult{}, nil
}
