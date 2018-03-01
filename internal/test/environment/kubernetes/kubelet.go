package kubernetes

import (
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
)

func kubeletContainerConfig(image, kubeconfigFileName, additionalArgs string) *container.Config {
	cmd := []string{"kubelet",
		"--allow-privileged=true",
		"--cadvisor-port=0",
		"--cluster-dns=10.96.0.10",
		"--cluster-domain=cluster.local",
		"--fail-swap-on=false",
		fmt.Sprintf("--kubeconfig=/kube-config/%s", kubeconfigFileName),
		"--require-kubeconfig=true",
		"--register-node",
		"--node-labels", "node-role.kubernetes.io/master=",
		additionalArgs,
	}

	// TODO: configure logging levels

	return &container.Config{
		Image: image,
		Cmd:   cmd,
	}
}

var kubeletHostConfig = &container.HostConfig{
	Privileged:  true,
	PidMode:     "host",
	IpcMode:     "host",
	NetworkMode: "host",
	RestartPolicy: container.RestartPolicy{
		Name: "always",
	},
	Mounts: []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: "/var/run/docker.sock",
			Target: "/var/run/docker.sock",
		},
		{
			Type:   mount.TypeBind,
			Source: "/var/lib/kubelet",
			Target: "/var/lib/kubelet",
			BindOptions: &mount.BindOptions{
				Propagation: mount.PropagationRShared,
			},
		},
		{
			Type:   mount.TypeBind,
			Source: "/var/lib/docker",
			Target: "/var/lib/docker",
		},
		{
			Type:   mount.TypeBind,
			Source: "/var/log/pods",
			Target: "/var/log/pods",
		},
		{
			Type:   mount.TypeBind,
			Source: "/var/log/containers",
			Target: "/var/log/containers",
		},
		{
			Type:   mount.TypeBind,
			Source: "/run",
			Target: "/run",
			BindOptions: &mount.BindOptions{
				Propagation: mount.PropagationRShared,
			},
		},
		{
			Type:   mount.TypeBind,
			Source: "/etc/cni",
			Target: "/etc/cni",
		},
		{
			Type:   mount.TypeBind,
			Source: "/opt/cni",
			Target: "/opt/cni",
		},
		{
			Type:   mount.TypeVolume,
			Source: "kube-config",
			Target: "/kube-config",
		},
	},
}
