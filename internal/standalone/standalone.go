// Package standalone implements a daemonless backend for the Docker CLI.
//
// When enabled (DOCKER_STANDALONE=1), the CLI does not talk to a Docker
// Engine over the API. Instead it embeds the containerd libraries (content
// store, metadata store, snapshotters, image puller, OCI spec generator and
// the runtime v2 shim manager) and drives an OCI runtime (crun or runc)
// directly. No dockerd or containerd daemon is required; the only
// per-container helper processes are containerd-shim-runc-v2 (which owns
// the container's I/O and lifecycle, exactly as it does under containerd) and
// a logging process provided by this binary.
//
// The backend supports running as root as well as rootless. In rootless mode
// the CLI re-executes itself inside a new user and mount namespace using the
// caller's /etc/subuid and /etc/subgid ranges (via newuidmap/newgidmap), and
// containers are networked with pasta.
package standalone

import (
	"os"
	"strconv"
)

const (
	// EnvEnabled is the environment variable that enables standalone mode.
	EnvEnabled = "DOCKER_STANDALONE"
	// EnvRoot overrides the directory used for persistent state (images,
	// containers, volumes).
	EnvRoot = "DOCKER_STANDALONE_ROOT"
	// EnvRunDir overrides the directory used for transient state (shim
	// sockets, bundles, FIFOs, network namespaces).
	EnvRunDir = "DOCKER_STANDALONE_RUN"
	// EnvRuntime selects the OCI runtime binary (crun, runc, or a path).
	EnvRuntime = "DOCKER_STANDALONE_RUNTIME"
	// EnvSnapshotter selects the containerd snapshotter (overlayfs, native).
	EnvSnapshotter = "DOCKER_STANDALONE_SNAPSHOTTER"
	// EnvNetwork selects the default network backend (bridge, pasta,
	// slirp4netns, host, none).
	EnvNetwork = "DOCKER_STANDALONE_NETWORK"

	// loggerArg is the first argument the binary is invoked with by the
	// shim when acting as a container logging process.
	loggerArg = "_standalone-logger"
)

// Enabled reports whether standalone mode was requested through the
// DOCKER_STANDALONE environment variable.
func Enabled() bool {
	v, ok := os.LookupEnv(EnvEnabled)
	if !ok || v == "" {
		return false
	}
	if b, err := strconv.ParseBool(v); err == nil {
		return b
	}
	return true
}

// IsLoggerInvocation reports whether the process was started by the shim as
// the per-container logging helper. Callers must invoke [RunLogger] in that
// case instead of running the CLI.
func IsLoggerInvocation(args []string) bool {
	return len(args) >= 3 && args[1] == loggerArg
}
