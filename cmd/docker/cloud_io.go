package main

import (
	"os"
	"os/exec"
	"runtime"

	"github.com/docker/cli/cli/command"
)

func configureCloudResolverIO(cmd *exec.Cmd, dockerCli *command.DockerCli) {
	cmd.Stderr = dockerCli.Err()
	if !dockerCli.In().IsTerminal() || !dockerCli.Err().IsTerminal() {
		return
	}

	stdin, stdinFile := dockerCli.In().File()
	stderr, stderrFile := dockerCli.Err().File()
	if runtime.GOOS == "windows" {
		// term.StdStreams can wrap console handles for terminal emulation,
		// preventing File from exposing them to the child process.
		if !stdinFile {
			stdin, stdinFile = os.Stdin, true
		}
		if !stderrFile {
			stderr, stderrFile = os.Stderr, true
		}
	}
	if stdinFile && stderrFile {
		// Pass files directly: wrapping them makes os/exec copy through pipes,
		// hiding terminal identity and potentially consuming the command's input.
		cmd.Stdin = stdin
		cmd.Stderr = stderr
	}
}
