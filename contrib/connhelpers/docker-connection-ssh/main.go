package main

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"

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
	if u.Scheme != "ssh" {
		return errors.Errorf("expected scheme: ssh, got %s", u.Scheme)
	}
	var (
		user   string
		host   string
		port   string
		socket = "/var/run/docker.sock"
	)

	if u.User != nil {
		user = u.User.Username()
		if _, ok := u.User.Password(); ok {
			return errors.New("ssh does not accept plain-text password")
		}
	}
	host = u.Hostname()
	port = u.Port()
	if u.Path != "" {
		socket = u.Path
	}
	return execSSH(user, host, port, socket)
}

func execSSH(user, host, port, socket string) error {
	var args []string
	if user != "" {
		args = append(args, "-l", user)
	}
	if port != "" {
		args = append(args, "-p", port)
	}
	// TODO: use "docker run alpine/socat" when socat is not installed?
	args = append(args, host, "--",
		"socat", "unix:"+socket, "stdio")
	cmd := exec.Command("ssh", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
