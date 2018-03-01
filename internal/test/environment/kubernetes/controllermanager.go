package kubernetes

import (
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
)

func controllerManagerContainerConfig(image string) *container.Config {
	cmd := []string{"kube-controller-manager",
		"--address=127.0.0.1",
		"--kubeconfig=/kube-config/controller-manager.config",
		"--leader-elect=true",
		"--root-ca-file=/kube-config/ca.crt",
		"--service-account-private-key-file=/kube-config/sa-signing.key",
		"--cluster-signing-cert-file=/kube-config/ca.crt",
		"--cluster-signing-key-file=/kube-config/ca.key",
		"--use-service-account-credentials=true",
		"--controllers=*,bootstrapsigner,tokencleaner",
		"--allocate-node-cidrs=true",
		"--cluster-cidr=10.244.0.0/16",
	}

	// TODO: configure logging levels

	return &container.Config{
		Image: image,
		Cmd:   cmd,
		Healthcheck: &container.HealthConfig{
			Test: []string{
				"CMD-SHELL",
				fmt.Sprintf("wget -O - -T %d -q --content-on-error http://127.0.0.1:10252/healthz || exit 1",
					3),
			},
			Interval: 3 * time.Second,
			Timeout:  3 * time.Second,
			Retries:  1,
		},
	}
}

var controllerManagerHostConfig = &container.HostConfig{
	RestartPolicy: container.RestartPolicy{
		Name: "always",
	},
	Mounts: []mount.Mount{
		{
			Type:   mount.TypeVolume,
			Source: "kube-config",
			Target: "/kube-config",
		},
	},
}
