package kubernetes

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	volumetypes "github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/gotestyourself/gotestyourself/poll"
)

const (
	// FIXME(vdemeester) those are prefect
	hyperkubeImage = "gcr.io/google-containers/hyperkube-amd64:v1.8.1"
	etcdImage      = "quay.io/coreos/etcd:v3.2.9"
	busyboxImage   = "busybox:latest"
)

// Setup start a fully containerized kubernetes cluster for e2e tests
func Setup(ctx context.Context, w io.Writer, kubeconfig string) error {
	c, err := client.NewEnvClient()
	if err != nil {
		return err
	}

	dinf, err := c.Info(ctx)
	if err != nil {
		return err
	}
	hostname := dinf.Name

	fmt.Fprintln(w, "Generating certificate authority and kube certificates...")
	ca, certs, err := generateCertificates(hostname)
	if err != nil {
		return err
	}

	fmt.Fprintln(w, "Creating kube-config volume...")
	if err := createKubeConfigVolume(ctx, c, hostname, ca, certs); err != nil {
		return err
	}

	fmt.Fprintln(w, "Creating kube-system network...")
	n, err := c.NetworkCreate(ctx, "kube-system", types.NetworkCreate{})
	if err != nil {
		return err
	}

	starter := containerStarter{c: c, ctx: ctx, network: n.ID, w: w}
	if err := starter.startContainers([]containerSetup{
		{
			name:    "kube-kubelet",
			cfg:     kubeletContainerConfig(hyperkubeImage, "kubelet.config", ""),
			hostCfg: kubeletHostConfig,
		},
		{
			name:    "kube-etcd",
			cfg:     &container.Config{Image: etcdImage, Cmd: []string{"/usr/local/bin/etcd", "-advertise-client-urls=http://0.0.0.0:2379", "-listen-client-urls=http://0.0.0.0:2379"}},
			hostCfg: &container.HostConfig{RestartPolicy: container.RestartPolicy{Name: "always"}},
		},
		{
			name:    "kube-apiserver",
			cfg:     apiServerContainerConfig(hyperkubeImage, ""),
			hostCfg: apiServerHostConfig,
		},
		{
			name:    "kube-controller-manager",
			cfg:     controllerManagerContainerConfig(hyperkubeImage),
			hostCfg: controllerManagerHostConfig,
		},
		{
			name:    "kube-scheduler",
			cfg:     schedulerContainerConfig(hyperkubeImage),
			hostCfg: schedulerHostConfig,
		},
	}); err != nil {
		return err
	}

	fmt.Fprintln(w, "Write kubeconfig file...")
	if err := writeKubeConfig(hostname, kubeconfig, ca); err != nil {
		return err
	}

	t := &fakeT{}
	fmt.Fprintln(w, "Starting kube-dns...")
	poll.WaitOn(t, applyYaml(ctx, kubeconfig, kubeDNSYaml), poll.WithDelay(500*time.Millisecond), poll.WithTimeout(15*time.Second))

	return starter.startContainer("kube-proxy", proxyContainerConfig(hyperkubeImage), proxyHostConfig)
}

func writeKubeConfig(hostname, kubeconfig string, ca *certAuthority) error {
	kubecfg, err := makeKubeConfig(kubeConfigSpec{
		clientName:    "kubernetes-admin",
		organization:  []string{"system:masters"},
		apiserverHost: hostname,
	}, ca)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(kubeconfig, []byte(kubecfg), 0644)
}

func createKubeConfigVolume(ctx context.Context, c client.APIClient, hostname string, ca *certAuthority, certs map[string]cert) error {
	configSpecs := map[string]kubeConfigSpec{
		"admin.config": {
			clientName:    "kubernetes-admin",
			organization:  []string{"system:masters"},
			apiserverHost: "kube-apiserver",
		},
		"kubelet.config": {
			clientName:   "system:node:" + hostname,
			organization: []string{"system:nodes"},
		},
		"proxy.config": {
			clientName:   "kube-proxy",
			organization: []string{"system:node-proxier"},
		},
		"controller-manager.config": {
			clientName:    "system:kube-controller-manager",
			apiserverHost: "kube-apiserver",
		},
		"scheduler.config": {
			clientName:    "system:kube-scheduler",
			apiserverHost: "kube-apiserver",
		},
	}
	if _, err := c.VolumeCreate(ctx, volumetypes.VolumesCreateBody{Name: "kube-config"}); err != nil {
		return err
	}
	cont, err := c.ContainerCreate(ctx, &container.Config{
		Image: busyboxImage,
		Cmd:   []string{"top"},
	}, &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: "kube-config",
				Target: "/kube-config",
			},
		},
	}, nil, "kube-config-init")
	if err != nil {
		return err
	}
	defer c.ContainerRemove(ctx, cont.ID, types.ContainerRemoveOptions{Force: true})

	if err := c.ContainerStart(ctx, cont.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	configDir := filepath.Join(dir, "kube-config")
	if err := os.Mkdir(configDir, 0700); err != nil {
		return err
	}
	defer os.RemoveAll(configDir)

	if err := generateAndWriteCerts(configDir, certs); err != nil {
		return err
	}

	if err := generateKubeConfig(configDir, configSpecs, ca); err != nil {
		return err
	}

	return copyToContainer(ctx, cont.ID, configDir)
}

func generateAndWriteCerts(dir string, certs map[string]cert) error {
	saSigningKey, err := newPVK()
	if err != nil {
		return err
	}

	if err := writeCertAndKey(dir, "ca", certs["ca"].cert, certs["ca"].pvk); err != nil {
		return err
	}

	if err := writePublicAndPrivateKey(dir, "sa-signing", saSigningKey); err != nil {
		return err
	}

	for name, cert := range certs {
		if err := writeCertAndKey(dir, name, cert.cert, cert.pvk); err != nil {
			return err
		}
	}
	return nil
}

func generateKubeConfig(dir string, configSpecs map[string]kubeConfigSpec, ca *certAuthority) error {
	for k, v := range configSpecs {
		conf, err := makeKubeConfig(v, ca)
		if err != nil {
			return err
		}
		filename := filepath.Join(dir, k)
		if err := ioutil.WriteFile(filename, []byte(conf), 0700); err != nil {
			return err
		}
	}
	return nil
}

func copyToContainer(ctx context.Context, containerID, dir string) error {
	// FIXME(vdemeester) use API only ? :P
	cmd := exec.CommandContext(ctx, "docker", "cp", dir, containerID+":/")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func applyYaml(ctx context.Context, kubeconfig, yaml string) func(poll.LogT) poll.Result {
	return func(l poll.LogT) poll.Result {
		cmd := exec.CommandContext(ctx, "/usr/local/bin/kubectl",
			"--kubeconfig="+kubeconfig,
			"apply",
			"-f",
			"-",
		)
		cmd.Stdin = strings.NewReader(yaml)
		if err := cmd.Run(); err != nil {
			return poll.Continue("kubectl failed : %v", err.Error())
		}
		return poll.Success()
	}
}

type kubeConfigSpec struct {
	clientName    string
	organization  []string
	apiserverHost string
}

func makeKubeConfig(spec kubeConfigSpec, ca *certAuthority) (string, error) {
	host := spec.apiserverHost
	if host == "" {
		host = "127.0.0.1"
	}
	const tpl = `apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: %s
    server: https://%s:6443
  name: kubernetes
contexts:
- context:
    cluster: kubernetes
    user: %v
  name: u
current-context: u
kind: Config
preferences: {}
users:
- name: %v
  user:
    client-certificate-data: %v
    client-key-data: %v`
	clientCert, clientKey, err := ca.newSignedCert(spec.clientName, spec.organization, []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, nil, nil, 10*365*24*time.Hour)
	if err != nil {
		return "", err
	}
	caPEM := encodeCertPEM(ca.caCert)
	clientCertPEM := encodeCertPEM(clientCert)
	clientKeyPEM := encodePrivateKeyPEM(clientKey)
	return fmt.Sprintf(tpl,
		base64.StdEncoding.EncodeToString(caPEM),
		host,
		spec.clientName,
		spec.clientName,
		base64.StdEncoding.EncodeToString(clientCertPEM),
		base64.StdEncoding.EncodeToString(clientKeyPEM)), nil
}

type containerSetup struct {
	name    string
	cfg     *container.Config
	hostCfg *container.HostConfig
}
type containerStarter struct {
	network string
	c       client.APIClient
	ctx     context.Context
	w       io.Writer
}

func (s *containerStarter) startContainer(name string, cfg *container.Config, hostCfg *container.HostConfig) error {
	fmt.Fprintf(s.w, "Starting %s...\n", name)
	c, err := s.c.ContainerCreate(s.ctx, cfg, hostCfg, nil, name)
	if err != nil {
		return err
	}
	if hostCfg == nil || hostCfg.NetworkMode != "host" {
		err = s.c.NetworkConnect(s.ctx, s.network, c.ID, nil)
		if err != nil {
			return err
		}
	}
	return s.c.ContainerStart(s.ctx, c.ID, types.ContainerStartOptions{})
}

func (s *containerStarter) startContainers(containers []containerSetup) error {
	for _, c := range containers {
		err := s.startContainer(c.name, c.cfg, c.hostCfg)
		if err != nil {
			return err
		}
	}
	return nil
}

type fakeT struct {
	failed string
}

func (t *fakeT) Fatalf(format string, args ...interface{}) {
	t.failed = fmt.Sprintf(format, args...)
	panic("exit wait on")
}

func (t *fakeT) Log(args ...interface{}) {
	fmt.Fprint(os.Stderr, args...)
}

func (t *fakeT) Logf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
}
