package kubernetes

import (
	"fmt"
	"strconv"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
)

func apiServerContainerConfig(image, advertiseAddress string) *container.Config {
	portMap := make(nat.PortSet)
	portMap["6443/tcp"] = struct{}{}
	cmd := []string{"kube-apiserver",
		"--etcd-servers=http://kube-etcd:2379",
		"--insecure-port=0",
		"--admission-control=Initializers,NamespaceLifecycle,LimitRanger,ServiceAccount,PersistentVolumeLabel,DefaultStorageClass,DefaultTolerationSeconds,NodeRestriction,ResourceQuota",
		fmt.Sprintf("--service-cluster-ip-range=%s", "10.96.0.0/16"),
		"--service-account-key-file=/kube-config/sa-signing.pub",
		"--client-ca-file=/kube-config/ca.crt",
		"--tls-cert-file=/kube-config/apiserver.crt",
		"--tls-private-key-file=/kube-config/apiserver.key",
		"--kubelet-client-certificate=/kube-config/kubelet-client.crt",
		"--kubelet-client-key=/kube-config/kubelet-client.key",
		"--enable-bootstrap-token-auth=true",
		"--secure-port=6443",
		"--allow-privileged=true",
		"--kubelet-preferred-address-types=InternalIP,ExternalIP,Hostname",
		"--requestheader-username-headers=X-Remote-User",
		"--requestheader-group-headers=X-Remote-Group",
		"--requestheader-extra-headers-prefix=X-Remote-Extra-",
		"--requestheader-client-ca-file=/kube-config/front-ca.crt",
		"--requestheader-allowed-names=kube-front-client",
		"--proxy-client-cert-file=/kube-config/front-client.crt",
		"--proxy-client-key-file=/kube-config/front-client.key",
	}
	if advertiseAddress != "" {
		cmd = append(cmd, "--advertise-address="+advertiseAddress)
	}
	return &container.Config{
		Image:        image,
		Cmd:          cmd,
		ExposedPorts: portMap,
	}
}

var apiServerHostConfig = &container.HostConfig{
	Privileged: true,
	RestartPolicy: container.RestartPolicy{
		Name: "always",
	},
	PortBindings: nat.PortMap{
		nat.Port("6443/tcp"): []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: strconv.Itoa(6443),
			},
		},
	},
	Mounts: []mount.Mount{
		{
			Type:   mount.TypeVolume,
			Source: "kube-config",
			Target: "/kube-config",
		},
	},
}
