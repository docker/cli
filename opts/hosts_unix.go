// +build !windows

package opts

import (
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

var (
	chosenDefaultUnixSocket     = defaultUnixSocket
	chooseDefaultUnixSocketOnce sync.Once
)

func chooseDefaultUnixSocket() {
	euid := os.Geteuid()
	if euid == 0 {
		return
	}
	var userSocketCandidates []string
	if xrd, ok := os.LookupEnv("XDG_RUNTIME_DIR"); ok {
		userSocketCandidates = append(userSocketCandidates, filepath.Join(xrd, "docker.sock"))
	} else {
		userSocketCandidates = append(userSocketCandidates, filepath.Join("/run/user", strconv.Itoa(euid), "docker.sock"))
	}
	if u, err := user.Current(); err != nil && u.HomeDir != "" {
		// Used on non-systemd hosts. (See dockerd-rootless-setuptool.sh)
		userSocketCandidates = append(userSocketCandidates, filepath.Join(u.HomeDir, ".docker/run/docker.sock"))
	}
	rootSocketOk := isSocketAccessible(defaultUnixSocket)
	for _, userSocket := range userSocketCandidates {
		if userSocketOk := isSocketAccessible(userSocket); userSocketOk {
			if rootSocketOk {
				// For compatibility, rootful socket is prioritized over rootless socket.
				logrus.Warnf("Both rootful socket (%q) and rootless socket (%q) are accessible. Choosing the rootful socket as the default one.",
					defaultUnixSocket, userSocket)
				return
			}
			logrus.Debugf("Automatically chose the default socket %q (rootless)", userSocket)
			chosenDefaultUnixSocket = userSocket
			return
		}
	}
}

func isSocketAccessible(s string) bool {
	return unix.Access(s, unix.R_OK|unix.W_OK) == nil
}

// DefaultLocalHost returns "unix:///var/run/docker.sock" is in most cases.
//
// However, DefaultLocalHost may return the rootless socket if the rootless socket
// is accessible and the rootless socket is inaccessible.
//
// When both the rootful one and the rootless one are accessible, the rootful one
// is returned (for compatibility).
//
// The rootless socket is typically "unix:///run/user/$UID/docker.sock", but
// can be "unix://$HOME/.docker/run/docker.sock" on non-systemd hosts.
func DefaultLocalHost() string {
	chooseDefaultUnixSocketOnce.Do(chooseDefaultUnixSocket)
	return "unix://" + chosenDefaultUnixSocket
}

// defaultHTTPHost Default HTTP Host used if only port is provided to -H flag e.g. dockerd -H tcp://:8080
const defaultHTTPHost = "localhost"
