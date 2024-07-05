package app

import (
	"github.com/docker/docker/client"
)

type fakeClient struct {
	client.Client
}
