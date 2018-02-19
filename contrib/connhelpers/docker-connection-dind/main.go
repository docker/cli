package main

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

func main() {
	if err := xmain(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func xmain() error {
	if len(os.Args) != 3 || os.Args[1] != "connect" {
		return errors.Errorf("usage: %s connect URL", os.Args[0])
	}
	u, err := url.Parse(os.Args[2])
	if err != nil {
		return err
	}
	if u.Scheme != "dind" {
		return errors.Errorf("expected scheme: dind, got %s", u.Scheme)
	}
	// TODO: support setting realDockerHost and realDockerTLSVerify
	// dind://containername?host=tcp%3A%2F%2Fhost%3A2376?tlsVerify=1
	return execDockerExec(u.Hostname(), "", false)
}

func execDockerExec(containerName, realDockerHost string, realDockerTLSVerify bool) error {
	// Astonishngly we can't use nc, as nc does not exit when the remote connection is closed.
	// cmd := exec.Command("docker", "exec", "-i", containerName, "nc", "localhost", "2375")
	cmd := exec.Command("docker", "exec", "-i", containerName,
		"docker", "run", "-i", "--rm", "-v", "/var/run/docker.sock:/var/run/docker.sock", "alpine/socat:1.0.1", "unix:/var/run/docker.sock", "stdio")
	cmd.Env = os.Environ()
	for i, s := range cmd.Env {
		if strings.HasPrefix(s, "DOCKER_HOST=") || strings.HasPrefix(s, "DOCKER_TLS_VERIFY=") {
			cmd.Env = append(cmd.Env[:i], cmd.Env[i+1:]...)
		}
	}
	if realDockerHost != "" {
		cmd.Env = append(cmd.Env, "DOCKER_HOST="+realDockerHost)
	}
	if realDockerTLSVerify {
		cmd.Env = append(cmd.Env, "DOCKER_TLS_VERIFY=1")
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
