package command

import (
	"fmt"
	"os"
	"os/exec"
)

func RunSwarm(dockerCli Cli) error {
	fmt.Println("\x1b[1;31m    'Swarm' is a Mirantis product. You should use swarmctl to manage swarm cluster.\x1b[0m\n    https://github.com/moby/swarmctl\n")
	c := exec.Command("swarmctl", os.Args[1:]...)
	c.Stdout = dockerCli.Out()
	c.Stderr = dockerCli.Err()
	return c.Run()
}
