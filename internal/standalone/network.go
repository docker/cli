//go:build linux

package standalone

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	cerrdefs "github.com/containerd/errdefs"
	gocni "github.com/containerd/go-cni"
	"github.com/containerd/log"
	cnins "github.com/containernetworking/plugins/pkg/ns"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/sys/userns"
	"golang.org/x/sys/unix"
)

// Network backends. "bridge" is the Docker default network; depending on the
// host it is implemented with CNI (rootful, when the CNI plugins are
// installed) or with pasta (rootless, or when CNI is unavailable).
const (
	networkBridge = "bridge"
	networkPasta  = "pasta"
	networkSlirp  = "slirp4netns"
	networkHost   = "host"
	networkNone   = "none"

	// Addresses used by pasta for DNS forwarding and to reach the host.
	pastaDNSForward   = "169.254.1.1"
	pastaMapGuestAddr = "169.254.1.2"

	cniBridgeName = "docker-sa0"
	cniSubnet     = "10.88.0.0/16"
	cniGateway    = "10.88.0.1"
)

var cniPluginDirs = []string{"/opt/cni/bin", "/usr/lib/cni", "/usr/libexec/cni", "/usr/local/lib/cni"}

// cniPluginDir returns the directory containing the bridge CNI plugin, if any.
func cniPluginDir() string {
	for _, d := range cniPluginDirs {
		if _, err := os.Stat(filepath.Join(d, "bridge")); err == nil {
			return d
		}
	}
	return ""
}

func hasBinary(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// defaultNetworkBackend picks the backend implementing the "bridge" network.
func defaultNetworkBackend(cfg *config) string {
	if !cfg.rootless && cniPluginDir() != "" {
		return networkBridge
	}
	if hasBinary("pasta") {
		return networkPasta
	}
	if hasBinary("slirp4netns") {
		return networkSlirp
	}
	return networkHost
}

// resolveNetworkBackend maps the container's --network setting to a backend.
func (e *engine) resolveNetworkBackend(hc *container.HostConfig) (string, error) {
	mode := networkModeName(hc)
	switch mode {
	case networkBridge:
		return e.cfg.network, nil
	case networkHost, networkNone:
		return mode, nil
	case networkPasta, networkSlirp:
		if !hasBinary(mode) {
			return "", fmt.Errorf("network %q requires the %s binary: %w", mode, mode, cerrdefs.ErrNotFound)
		}
		return mode, nil
	}
	if hc.NetworkMode.IsContainer() {
		return "", fmt.Errorf("--network=container:<id> is not supported in standalone mode: %w", cerrdefs.ErrNotImplemented)
	}
	return "", fmt.Errorf("network %s not found: %w", mode, cerrdefs.ErrNotFound)
}

// ---- network namespaces

// createNetns creates a new, empty network namespace and bind-mounts it at
// path so that it outlives the creating thread.
func createNetns(path string) (retErr error) {
	// pasta drops privileges to "nobody" when started as root and must
	// still be able to open the namespace file, hence 0755.
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE|os.O_EXCL, 0o444)
	if err != nil {
		return err
	}
	_ = f.Close()
	defer func() {
		if retErr != nil {
			_ = os.Remove(path)
		}
	}()

	errCh := make(chan error, 1)
	go func() {
		// Never unlock: the thread is discarded when the goroutine exits.
		runtime.LockOSThread()
		orig, err := os.Open(fmt.Sprintf("/proc/self/task/%d/ns/net", unix.Gettid()))
		if err != nil {
			errCh <- err
			return
		}
		defer orig.Close()
		if err := unix.Unshare(unix.CLONE_NEWNET); err != nil {
			errCh <- fmt.Errorf("creating network namespace: %w", err)
			return
		}
		// Bind-mounting the namespace keeps it alive once no thread is in
		// it anymore, and gives the runtime a stable path to join.
		err = unix.Mount(fmt.Sprintf("/proc/self/task/%d/ns/net", unix.Gettid()), path, "none", unix.MS_BIND, "")
		_ = unix.Setns(int(orig.Fd()), unix.CLONE_NEWNET)
		if err != nil {
			errCh <- fmt.Errorf("persisting network namespace: %w", err)
			return
		}
		errCh <- nil
	}()
	return <-errCh
}

// removeNetns unmounts and removes a persisted network namespace. It is a
// no-op if the namespace is gone already.
func removeNetns(path string) error {
	if path == "" {
		return nil
	}
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if err := unix.Unmount(path, unix.MNT_DETACH); err != nil && !errors.Is(err, unix.EINVAL) && !errors.Is(err, unix.ENOENT) {
		return fmt.Errorf("unmounting netns %s: %w", path, err)
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// netnsAddresses returns the non-loopback IPv4 addresses configured inside
// the network namespace.
func netnsAddresses(nsPath string) (addrs []netip.Prefix, err error) {
	netNS, err := cnins.GetNS(nsPath)
	if err != nil {
		return nil, err
	}
	defer netNS.Close()
	err = netNS.Do(func(cnins.NetNS) error {
		ifaces, err := net.Interfaces()
		if err != nil {
			return err
		}
		for _, iface := range ifaces {
			if iface.Flags&net.FlagLoopback != 0 {
				continue
			}
			ifAddrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, a := range ifAddrs {
				ipn, ok := a.(*net.IPNet)
				if !ok {
					continue
				}
				ip, ok := netip.AddrFromSlice(ipn.IP)
				if !ok || !ip.Is4() && !ip.Unmap().Is4() {
					continue
				}
				ones, _ := ipn.Mask.Size()
				addrs = append(addrs, netip.PrefixFrom(ip.Unmap(), ones))
			}
		}
		return nil
	})
	return addrs, err
}

// ---- port mappings

// portMapping is a resolved published port.
type portMapping struct {
	proto         string
	hostIP        netip.Addr
	hostPort      uint16
	containerPort uint16
}

// resolvePortMappings expands PortBindings / PublishAllPorts into concrete
// host ports, allocating ephemeral ports where none was given.
func resolvePortMappings(cfg *container.Config, hc *container.HostConfig) ([]portMapping, network.PortMap, error) {
	var out []portMapping
	result := network.PortMap{}
	var ports []network.Port
	for p := range hc.PortBindings {
		ports = append(ports, p)
	}
	if hc.PublishAllPorts && cfg != nil {
		for p := range cfg.ExposedPorts {
			if _, ok := hc.PortBindings[p]; !ok {
				ports = append(ports, p)
			}
		}
	}
	sort.Slice(ports, func(i, j int) bool { return ports[i].String() < ports[j].String() })
	for _, p := range ports {
		bindings := hc.PortBindings[p]
		if len(bindings) == 0 {
			bindings = []network.PortBinding{{}}
		}
		for _, b := range bindings {
			hostPort, err := parseHostPort(b.HostPort, string(p.Proto()))
			if err != nil {
				return nil, nil, err
			}
			hostIP := b.HostIP
			out = append(out, portMapping{proto: string(p.Proto()), hostIP: hostIP, hostPort: hostPort, containerPort: p.Num()})
			if !hostIP.IsValid() {
				hostIP = netip.IPv4Unspecified()
			}
			result[p] = append(result[p], network.PortBinding{HostIP: hostIP, HostPort: strconv.Itoa(int(hostPort))})
		}
	}
	return out, result, nil
}

func parseHostPort(s, proto string) (uint16, error) {
	if s == "" {
		return allocatePort(proto)
	}
	if lo, _, ok := strings.Cut(s, "-"); ok {
		s = lo
	}
	n, err := strconv.ParseUint(s, 10, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid host port %q: %w", s, cerrdefs.ErrInvalidArgument)
	}
	return uint16(n), nil
}

// allocatePort asks the kernel for a free ephemeral port.
func allocatePort(proto string) (uint16, error) {
	if proto == "udp" {
		c, err := net.ListenPacket("udp", ":0")
		if err != nil {
			return 0, err
		}
		defer c.Close()
		return uint16(c.LocalAddr().(*net.UDPAddr).Port), nil
	}
	l, err := net.Listen("tcp", ":0") //nolint:gosec // G102: binding to all interfaces is how a free port is probed, as the daemon does
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return uint16(l.Addr().(*net.TCPAddr).Port), nil
}

// ---- setup / teardown

// netHelperPidFile is where the pid of the container's network helper
// (pasta or slirp4netns) is stored, so that it can be terminated when the
// container stops instead of waiting for it to notice that the network
// namespace is gone.
func (e *engine) netHelperPidFile(id string) string {
	return filepath.Join(e.cfg.containerRunDir(id), "network.pid")
}

// setupNetwork prepares networking for a container that is about to start.
// It creates the namespace, attaches the backend and returns the resulting
// state (IP address, DNS servers, netns path).
//
//nolint:gocyclo // one branch per supported network backend
func (e *engine) setupNetwork(ctx context.Context, id string, m *containerMeta, backend string) (ns networkState, retErr error) {
	ns.Backend = backend
	mappings, portMap, err := resolvePortMappings(m.Config, m.HostConfig)
	if err != nil {
		return ns, err
	}
	ns.PortMap = portMap

	switch backend {
	case networkHost:
		return ns, nil
	case networkNone:
		// An isolated namespace with loopback only.
	}

	nsPath := filepath.Join(e.cfg.netnsDir(), shortID(id))
	_ = removeNetns(nsPath)
	if err := createNetns(nsPath); err != nil {
		return ns, err
	}
	ns.NetnsPath = nsPath
	defer func() {
		if retErr != nil {
			_ = removeNetns(nsPath)
		}
	}()

	switch backend {
	case networkNone:
		return ns, nil
	case networkPasta:
		if err := setupPasta(ctx, nsPath, e.netHelperPidFile(id), mappings); err != nil {
			return ns, err
		}
		if pastaFeatures().dnsForward {
			ns.DNS = []string{pastaDNSForward}
		}
	case networkSlirp:
		if err := setupSlirp4netns(e.cfg, id, nsPath, e.netHelperPidFile(id), mappings); err != nil {
			return ns, err
		}
		ns.Gateway = "10.0.2.2"
		ns.DNS = []string{"10.0.2.3"}
	case networkBridge:
		res, err := e.setupCNI(ctx, id, nsPath, mappings)
		if err != nil {
			return ns, err
		}
		for _, cfg := range res.Interfaces {
			for _, ipcfg := range cfg.IPConfigs {
				if ipcfg.IP.To4() != nil {
					ns.IPAddress = ipcfg.IP.String()
					ns.Gateway = ipcfg.Gateway.String()
				}
			}
			if cfg.Mac != "" && !strings.HasPrefix(cfg.Mac, "00:00") {
				ns.MacAddress = cfg.Mac
			}
		}
		ns.IPPrefix = 16
		return ns, nil
	default:
		return ns, fmt.Errorf("unknown network backend %q: %w", backend, cerrdefs.ErrInvalidArgument)
	}

	addrs, err := netnsAddresses(nsPath)
	if err != nil {
		log.G(ctx).WithError(err).Debug("reading container addresses")
	}
	if len(addrs) > 0 {
		ns.IPAddress = addrs[0].Addr().String()
		ns.IPPrefix = addrs[0].Bits()
	}
	return ns, nil
}

// teardownNetwork releases network resources of a stopped container.
func (e *engine) teardownNetwork(ctx context.Context, id string, ns *networkState) {
	if ns == nil {
		return
	}
	// Stop the network helper first: it holds the container's published
	// ports, which the container must be able to publish again when it is
	// restarted.
	e.stopNetHelper(ctx, id)
	if ns.Backend == networkBridge && ns.NetnsPath != "" {
		if err := e.teardownCNI(ctx, id, ns.NetnsPath); err != nil {
			log.G(ctx).WithError(err).Debug("CNI teardown")
		}
	}
	if err := removeNetns(ns.NetnsPath); err != nil {
		log.G(ctx).WithError(err).Debug("removing network namespace")
	}
}

// stopNetHelper terminates the container's network helper and waits briefly
// for it to exit.
func (e *engine) stopNetHelper(ctx context.Context, id string) {
	data, err := os.ReadFile(e.netHelperPidFile(id))
	if err != nil {
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 1 {
		return
	}
	if err := unix.Kill(pid, unix.SIGTERM); err != nil {
		return
	}
	for range 50 {
		if err := unix.Kill(pid, 0); err != nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	log.G(ctx).WithField("pid", pid).Debug("network helper did not exit after SIGTERM")
	_ = unix.Kill(pid, unix.SIGKILL)
}

// ---- pasta

func setupPasta(ctx context.Context, nsPath, pidFile string, mappings []portMapping) error {
	args := []string{"--config-net", "--pid", pidFile}
	if os.Geteuid() == 0 && !userns.RunningInUserNS() {
		// Started by real root, pasta would drop to "nobody" and lose the
		// privileges needed to join the namespace we created for it.
		args = append(args, "--runas", "0")
	}
	for _, pm := range mappings {
		flag := "-t"
		if pm.proto == "udp" {
			flag = "-u"
		}
		spec := fmt.Sprintf("%d:%d", pm.hostPort, pm.containerPort)
		if pm.hostIP.IsValid() && !pm.hostIP.IsUnspecified() {
			spec = pm.hostIP.String() + "/" + spec
		}
		args = append(args, flag, spec)
	}
	feats := pastaFeatures()
	if feats.dnsForward {
		args = append(args, "--dns-forward", pastaDNSForward)
	}
	if feats.mapGuestAddr {
		args = append(args, "--map-guest-addr", pastaMapGuestAddr)
	}
	args = append(args,
		"-t", "none", "-u", "none", "-T", "none", "-U", "none",
		"--no-map-gw",
		"--quiet",
		"--netns", nsPath,
	)
	// "-t none" must not override explicit mappings: pasta uses the first
	// occurrence, so only add the "none" defaults when nothing was mapped.
	args = dedupePastaNone(args, mappings)
	cmd := exec.CommandContext(ctx, "pasta", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pasta failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	if len(out) > 0 {
		log.G(ctx).Debugf("pasta: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

type pastaFeatureSet struct {
	dnsForward   bool
	mapGuestAddr bool
}

// pastaFeatures probes the installed pasta for optional flags.
var pastaFeatures = sync.OnceValue(func() pastaFeatureSet {
	out, _ := exec.Command("pasta", "--help").CombinedOutput()
	help := string(out)
	return pastaFeatureSet{
		dnsForward:   strings.Contains(help, "--dns-forward"),
		mapGuestAddr: strings.Contains(help, "--map-guest-addr"),
	}
})

func dedupePastaNone(args []string, mappings []portMapping) []string {
	hasTCP, hasUDP := false, false
	for _, pm := range mappings {
		if pm.proto == "udp" {
			hasUDP = true
		} else {
			hasTCP = true
		}
	}
	var out []string
	for i := 0; i < len(args); i++ {
		if i+1 < len(args) && args[i+1] == "none" && ((args[i] == "-t" && hasTCP) || (args[i] == "-u" && hasUDP)) {
			i++
			continue
		}
		out = append(out, args[i])
	}
	return out
}

// ---- slirp4netns

func setupSlirp4netns(cfg *config, id, nsPath, pidFile string, mappings []portMapping) error {
	if len(mappings) > 0 {
		return fmt.Errorf("published ports are not supported with the slirp4netns backend; use pasta: %w", cerrdefs.ErrNotImplemented)
	}
	readyR, readyW, err := os.Pipe()
	if err != nil {
		return err
	}
	defer readyR.Close()
	cmd := exec.Command("slirp4netns", "--configure", "--mtu=65520", "--disable-host-loopback", "-r", "3", "--netns-type=path", nsPath, "tap0")
	cmd.ExtraFiles = []*os.File{readyW}
	cmd.SysProcAttr = &unix.SysProcAttr{Setsid: true}
	logFile, err := os.OpenFile(filepath.Join(cfg.containerRunDir(id), "slirp4netns.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err == nil {
		cmd.Stdout, cmd.Stderr = logFile, logFile
		defer logFile.Close()
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting slirp4netns: %w", err)
	}
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0o600); err != nil {
		return err
	}
	_ = readyW.Close()
	go func() { _ = cmd.Wait() }()
	buf := make([]byte, 1)
	if _, err := readyR.Read(buf); err != nil {
		return fmt.Errorf("slirp4netns did not become ready: %w", err)
	}
	return nil
}

// ---- CNI (rootful bridge)

func (*engine) cniConfList() []byte {
	plugins := []map[string]any{
		{
			"type":        "bridge",
			"bridge":      cniBridgeName,
			"isGateway":   true,
			"ipMasq":      true,
			"hairpinMode": true,
			"ipam": map[string]any{
				"type": "host-local",
				"ranges": [][]map[string]string{{{
					"subnet":  cniSubnet,
					"gateway": cniGateway,
				}}},
				"routes": []map[string]string{{"dst": "0.0.0.0/0"}},
			},
		},
		{"type": "portmap", "capabilities": map[string]bool{"portMappings": true}},
	}
	dir := cniPluginDir()
	if _, err := os.Stat(filepath.Join(dir, "firewall")); err == nil {
		plugins = append(plugins, map[string]any{"type": "firewall"})
	}
	conf := map[string]any{
		"cniVersion": "1.0.0",
		"name":       "docker-standalone",
		"plugins":    plugins,
	}
	data, err := json.MarshalIndent(conf, "", "  ")
	if err != nil {
		// The configuration is a fixed literal; it cannot fail to encode.
		panic(err)
	}
	return data
}

func (e *engine) cni() (gocni.CNI, error) {
	dir := cniPluginDir()
	if dir == "" {
		return nil, fmt.Errorf("bridge networking requires the CNI plugins (bridge, host-local, portmap) in one of %s: %w", strings.Join(cniPluginDirs, ", "), cerrdefs.ErrNotFound)
	}
	c, err := gocni.New(gocni.WithMinNetworkCount(2), gocni.WithPluginDir([]string{dir}))
	if err != nil {
		return nil, err
	}
	if err := c.Load(gocni.WithLoNetwork, gocni.WithConfListBytes(e.cniConfList())); err != nil {
		return nil, fmt.Errorf("loading CNI configuration: %w", err)
	}
	return c, nil
}

func (e *engine) setupCNI(ctx context.Context, id, nsPath string, mappings []portMapping) (*gocni.Result, error) {
	c, err := e.cni()
	if err != nil {
		return nil, err
	}
	var pms []gocni.PortMapping
	for _, pm := range mappings {
		hostIP := ""
		if pm.hostIP.IsValid() && !pm.hostIP.IsUnspecified() {
			hostIP = pm.hostIP.String()
		}
		pms = append(pms, gocni.PortMapping{HostPort: int32(pm.hostPort), ContainerPort: int32(pm.containerPort), Protocol: pm.proto, HostIP: hostIP})
	}
	return c.Setup(ctx, id, nsPath, gocni.WithCapabilityPortMap(pms))
}

func (e *engine) teardownCNI(ctx context.Context, id, nsPath string) error {
	c, err := e.cni()
	if err != nil {
		return err
	}
	return c.Remove(ctx, id, nsPath)
}

// ---- API: networks

var builtinNetworks = []string{networkBridge, networkHost, networkNone}

func (e *engine) networkSummary(name string) network.Summary {
	driver := name
	if name == networkBridge {
		driver = e.cfg.network
	}
	n := network.Network{
		Name:       name,
		ID:         "standalone-" + name,
		Driver:     driver,
		Scope:      "local",
		EnableIPv4: true,
		Labels:     map[string]string{},
		Options:    map[string]string{},
	}
	if name == networkBridge && driver == networkBridge {
		n.IPAM = network.IPAM{Driver: "default", Config: []network.IPAMConfig{{
			Subnet:  netip.MustParsePrefix(cniSubnet),
			Gateway: netip.MustParseAddr(cniGateway),
		}}}
	}
	return network.Summary{Network: n}
}
