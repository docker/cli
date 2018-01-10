package kubernetes

import (
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
)

func proxyContainerConfig(image string) *container.Config {
	return &container.Config{
		Image: image,
		Cmd: []string{"kube-proxy",
			"--kubeconfig=/kube-config/proxy.config",
			"--v=2", // TODO: wire up log levels
		},
	}
}

var proxyHostConfig = &container.HostConfig{
	Privileged:  true,
	NetworkMode: "host",
	RestartPolicy: container.RestartPolicy{
		Name: "always",
	},
	Mounts: []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: "/run/xtables.lock",
			Target: "/run/xtables.lock",
		},
		{
			Type:   mount.TypeVolume,
			Source: "kube-config",
			Target: "/kube-config",
		},
	},
}
