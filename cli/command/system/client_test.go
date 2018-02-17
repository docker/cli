package system

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

type fakeClient struct {
	client.Client

	version       string
	serverVersion func(ctx context.Context) (types.Version, error)
}

func (cli *fakeClient) ServerVersion(ctx context.Context) (types.Version, error) {
	return cli.serverVersion(ctx)
}

func (cli *fakeClient) ClientVersion() string {
	return cli.version
}
