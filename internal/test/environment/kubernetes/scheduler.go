package kubernetes

import (
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
)

func schedulerContainerConfig(image string) *container.Config {
	cmd := []string{"kube-scheduler",
		"--address=127.0.0.1",
		"--kubeconfig=/kube-config/scheduler.config",
		"--leader-elect=true",
	}

	// TODO: configure logging levels

	return &container.Config{
		Image: image,
		Cmd:   cmd,
		Healthcheck: &container.HealthConfig{
			Test: []string{
				"CMD-SHELL",
				fmt.Sprintf("wget -O - -T %d -q --content-on-error http://127.0.0.1:10251/healthz || exit 1",
					3),
			},
			Interval: 3 * time.Second,
			Timeout:  3 * time.Second,
			Retries:  1,
		},
	}
}

var schedulerHostConfig = &container.HostConfig{
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
