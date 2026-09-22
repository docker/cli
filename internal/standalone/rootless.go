//go:build linux

package standalone

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/moby/sys/userns"
	"golang.org/x/sys/unix"
)

// Rootless mode.
//
// When the CLI runs as an unprivileged user, it re-executes itself inside a
// new user namespace (with the caller's /etc/subuid and /etc/subgid ranges
// mapped through newuidmap/newgidmap) and a new mount namespace. Inside that
// namespace the process is root and can mount image layers, create network
// namespaces and run the OCI runtime, while remaining unprivileged on the
// host. The shim, logging helper and pasta processes spawned for a container
// inherit the namespaces and keep them alive after the CLI exits; subsequent
// CLI invocations create fresh namespaces with identical ID mappings, so
// files and sockets remain accessible.
//
// The re-exec happens in two stages because the ID mappings must be written
// by the parent before the child executes as root:
//
//  1. the parent clones the child with CLONE_NEWUSER|CLONE_NEWNS and the
//     child (still unmapped, without capabilities) blocks on a pipe;
//  2. the parent writes the mappings and signals the pipe; the child then
//     re-executes itself, now as UID 0 in the namespace with full
//     capabilities, and runs the CLI.

const (
	envStage      = "_DOCKER_STANDALONE_STAGE"
	stageWait     = "wait"
	stageChild    = "child"
	syncFD        = 3
	subIDFallback = 65536
)

// MaybeReexecRootless re-executes the CLI inside a user namespace when
// standalone mode is enabled and the process is not privileged. It does not
// return when a re-exec happens (the process exits with the child's status);
// it returns nil when the current process should continue running the CLI.
func MaybeReexecRootless() error {
	if !Enabled() {
		return nil
	}
	switch os.Getenv(envStage) {
	case stageWait:
		waitForMappingsAndExec()
	case stageChild:
		_ = os.Unsetenv(envStage)
		cfg := &config{}
		cfg.runDir = os.Getenv(EnvRunDir)
		if cfg.runDir == "" {
			cfg.runDir = filepath.Join(runtimeDir(), "docker-standalone")
		}
		return setupChildMountNamespace(cfg.shimSocketDir())
	}
	if os.Geteuid() == 0 || userns.RunningInUserNS() {
		return nil
	}
	code, err := reexecInUserNamespace()
	if err != nil {
		return err
	}
	os.Exit(code)
	return nil
}

func reexecInUserNamespace() (int, error) {
	self, err := os.Executable()
	if err != nil {
		return 1, err
	}
	r, w, err := os.Pipe()
	if err != nil {
		return 1, err
	}
	defer r.Close()

	cmd := exec.Command(self, os.Args[1:]...)
	cmd.Env = append(os.Environ(), envStage+"="+stageWait)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.ExtraFiles = []*os.File{r}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUSER | syscall.CLONE_NEWNS,
	}
	if err := cmd.Start(); err != nil {
		_ = w.Close()
		return 1, fmt.Errorf("creating user namespace: %w", err)
	}
	pid := cmd.Process.Pid

	if err := writeIDMappings(pid); err != nil {
		_ = w.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return 1, err
	}
	if _, err := w.Write([]byte{'1'}); err != nil {
		return 1, err
	}
	_ = w.Close()

	// Forward terminal and termination signals to the child.
	sigCh := make(chan os.Signal, 16)
	signal.Notify(sigCh, unix.SIGINT, unix.SIGTERM, unix.SIGHUP, unix.SIGQUIT, unix.SIGWINCH, unix.SIGUSR1, unix.SIGUSR2)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case sig := <-sigCh:
				_ = cmd.Process.Signal(sig)
			case <-done:
				return
			}
		}
	}()
	err = cmd.Wait()
	close(done)
	signal.Stop(sigCh)
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
				return 128 + int(ws.Signal()), nil
			}
			return exitErr.ExitCode(), nil
		}
		return 1, err
	}
	return 0, nil
}

// waitForMappingsAndExec is the intermediate stage: block until the parent
// has written the ID mappings, then re-execute as root in the namespace.
func waitForMappingsAndExec() {
	buf := make([]byte, 1)
	n, _ := unix.Read(syncFD, buf)
	_ = unix.Close(syncFD)
	if n != 1 {
		fmt.Fprintln(os.Stderr, "docker: failed to set up user namespace mappings")
		os.Exit(1)
	}
	self, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "docker:", err)
		os.Exit(1)
	}
	env := []string{envStage + "=" + stageChild}
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, envStage+"=") {
			env = append(env, kv)
		}
	}
	if err := unix.Exec(self, os.Args, env); err != nil {
		fmt.Fprintln(os.Stderr, "docker: re-exec failed:", err)
		os.Exit(1)
	}
}

// setupChildMountNamespace prepares the new mount namespace:
//
//   - mounts become slaves of the host's, so that mounts performed by the
//     engine never propagate back;
//   - /run is replaced by a private tmpfs with the host's entries bound into
//     it (like rootlesskit --copy-up=/run), and the engine's shim socket
//     directory is bound at /run/containerd/s. Shims older than containerd
//     2.4 hardcode that path, which is not writable by an unprivileged
//     user on the host; binding a persistent directory there keeps the
//     sockets reachable from later CLI invocations (which get their own
//     mount namespace).
func setupChildMountNamespace(shimSocketDir string) error {
	if err := unix.Mount("", "/", "", unix.MS_SLAVE|unix.MS_REC, ""); err != nil {
		return fmt.Errorf("setting mount propagation: %w", err)
	}
	if err := os.MkdirAll(shimSocketDir, 0o700); err != nil {
		return err
	}
	if err := copyUpRun(); err != nil {
		return fmt.Errorf("preparing /run: %w", err)
	}
	if err := os.MkdirAll("/run/containerd/s", 0o700); err != nil {
		return err
	}
	if err := unix.Mount(shimSocketDir, "/run/containerd/s", "none", unix.MS_BIND, ""); err != nil {
		return fmt.Errorf("binding shim socket directory: %w", err)
	}
	return nil
}

// copyUpRun mounts a tmpfs over /run and binds the host's top-level entries
// into it, except /run/containerd which the engine manages itself.
func copyUpRun() error {
	orig, err := os.Open("/run")
	if err != nil {
		return err
	}
	defer orig.Close()
	entries, err := orig.Readdir(-1)
	if err != nil {
		return err
	}
	if err := unix.Mount("tmpfs", "/run", "tmpfs", unix.MS_NOSUID|unix.MS_NODEV, "mode=755"); err != nil {
		return fmt.Errorf("mounting tmpfs on /run: %w", err)
	}
	for _, fi := range entries {
		name := fi.Name()
		if name == "containerd" {
			continue
		}
		src := fmt.Sprintf("/proc/self/fd/%d/%s", orig.Fd(), name)
		dst := filepath.Join("/run", name)
		switch {
		case fi.Mode()&os.ModeSymlink != 0:
			target, err := os.Readlink(src)
			if err != nil {
				continue
			}
			_ = os.Symlink(target, dst)
		case fi.IsDir():
			if err := os.Mkdir(dst, fi.Mode().Perm()); err != nil {
				continue
			}
			_ = unix.Mount(src, dst, "none", unix.MS_BIND|unix.MS_REC, "")
		case fi.Mode().IsRegular():
			f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY, fi.Mode().Perm())
			if err != nil {
				continue
			}
			_ = f.Close()
			_ = unix.Mount(src, dst, "none", unix.MS_BIND, "")
		}
	}
	return nil
}

// writeIDMappings maps the current user to root and the user's subordinate
// ID ranges to 1..N in the child's namespace. Without subordinate IDs a
// single-ID mapping is used (which breaks images that need other UIDs).
func writeIDMappings(pid int) error {
	uid, gid := os.Getuid(), os.Getgid()
	uidRanges := subIDRanges("/etc/subuid", uid)
	gidRanges := subIDRanges("/etc/subgid", uid)
	newuidmap, err1 := exec.LookPath("newuidmap")
	newgidmap, err2 := exec.LookPath("newgidmap")

	if len(uidRanges) > 0 && len(gidRanges) > 0 && err1 == nil && err2 == nil {
		if err := runIDMap(newuidmap, pid, uid, uidRanges); err != nil {
			return fmt.Errorf("newuidmap: %w", err)
		}
		if err := runIDMap(newgidmap, pid, gid, gidRanges); err != nil {
			return fmt.Errorf("newgidmap: %w", err)
		}
		return nil
	}

	fmt.Fprintln(os.Stderr, "WARNING: no subordinate UID/GID ranges found for the current user "+
		"(check /etc/subuid, /etc/subgid and newuidmap/newgidmap); using a single-ID mapping, "+
		"which may break images that use other users")
	if err := os.WriteFile(fmt.Sprintf("/proc/%d/setgroups", pid), []byte("deny"), 0); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.WriteFile(fmt.Sprintf("/proc/%d/uid_map", pid), []byte(fmt.Sprintf("0 %d 1\n", uid)), 0); err != nil {
		return fmt.Errorf("writing uid_map: %w", err)
	}
	if err := os.WriteFile(fmt.Sprintf("/proc/%d/gid_map", pid), []byte(fmt.Sprintf("0 %d 1\n", gid)), 0); err != nil {
		return fmt.Errorf("writing gid_map: %w", err)
	}
	return nil
}

type idRange struct {
	start, count int
}

// subIDRanges reads the subordinate ID ranges for the user from a
// /etc/subuid-style file, matching by user name or numeric UID.
func subIDRanges(path string, uid int) []idRange {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	var names []string
	if u, err := user.LookupId(strconv.Itoa(uid)); err == nil {
		names = append(names, u.Username)
	}
	names = append(names, strconv.Itoa(uid))
	var ranges []idRange
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		parts := strings.Split(strings.TrimSpace(sc.Text()), ":")
		if len(parts) != 3 {
			continue
		}
		match := false
		for _, n := range names {
			if parts[0] == n {
				match = true
			}
		}
		if !match {
			continue
		}
		start, err1 := strconv.Atoi(parts[1])
		count, err2 := strconv.Atoi(parts[2])
		if err1 != nil || err2 != nil || count <= 0 {
			continue
		}
		ranges = append(ranges, idRange{start: start, count: count})
	}
	return ranges
}

// runIDMap invokes newuidmap/newgidmap: ID 0 maps to the caller, and the
// subordinate ranges are mapped contiguously from 1.
func runIDMap(tool string, pid, hostID int, ranges []idRange) error {
	args := make([]string, 0, 4+3*len(ranges))
	args = append(args, strconv.Itoa(pid), "0", strconv.Itoa(hostID), "1")
	next := 1
	for _, r := range ranges {
		args = append(args, strconv.Itoa(next), strconv.Itoa(r.start), strconv.Itoa(r.count))
		next += r.count
	}
	out, err := exec.Command(tool, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
