package system

import (
	"github.com/docker/docker/client"
)

type fakeClient struct {
	client.Client

	version string
}

func (cli *fakeClient) ClientVersion() string {
	return cli.version
}
