//go:build linux

package standalone

import (
	"bufio"
	"os"
	"strconv"
	"strings"

	ctrdversion "github.com/containerd/containerd/v2/version"
	"golang.org/x/sys/unix"
)

func containerdVersion() string {
	return ctrdversion.Version
}

func cgroupVersion() string {
	var st unix.Statfs_t
	if err := unix.Statfs("/sys/fs/cgroup", &st); err == nil && st.Type == unix.CGROUP2_SUPER_MAGIC {
		return "2"
	}
	return "1"
}

func isCgroup2() bool { return cgroupVersion() == "2" }

func osPrettyName() string {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return "Linux"
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if v, ok := strings.CutPrefix(sc.Text(), "PRETTY_NAME="); ok {
			return strings.Trim(v, `"`)
		}
	}
	return "Linux"
}

func totalMemory() int64 {
	var si unix.Sysinfo_t
	if err := unix.Sysinfo(&si); err != nil {
		return 0
	}
	return int64(si.Totalram) * int64(si.Unit)
}

// hostNameservers returns the nameservers from the host's resolv.conf,
// following the systemd-resolved stub to the real upstream servers.
func hostResolvConf() (nameservers, search, options []string) {
	paths := []string{"/etc/resolv.conf"}
	ns, se, op := parseResolvConf(paths[0])
	if len(ns) == 1 && (ns[0] == "127.0.0.53" || ns[0] == "127.0.0.54") {
		if n2, s2, o2 := parseResolvConf("/run/systemd/resolve/resolv.conf"); len(n2) > 0 {
			return n2, s2, o2
		}
	}
	return ns, se, op
}

func parseResolvConf(path string) (nameservers, search, options []string) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, nil
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || line[0] == '#' || line[0] == ';' {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "nameserver":
			nameservers = append(nameservers, fields[1])
		case "search":
			search = fields[1:]
		case "options":
			options = fields[1:]
		}
	}
	return nameservers, search, options
}

func isLoopbackAddr(addr string) bool {
	return strings.HasPrefix(addr, "127.") || addr == "::1"
}

func parseIntDefault(s string, def int) int {
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return def
}
