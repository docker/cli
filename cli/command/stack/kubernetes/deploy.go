package kubernetes

import (
	"fmt"

	"github.com/docker/cli/cli/command/stack/loader"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/pkg/errors"
)

// RunDeploy is the kubernetes implementation of docker stack deploy
func RunDeploy(dockerCli *KubeCli, opts options.Deploy) error {
	cmdOut := dockerCli.Out()
	// Check arguments
	if len(opts.Composefiles) == 0 {
		return errors.Errorf("Please specify only one compose file (with --compose-file).")
	}

	// Parse the compose file
	cfg, err := loader.LoadComposefile(dockerCli, opts)
	if err != nil {
		return err
	}
	stack, err := loadStack(opts.Namespace, *cfg)
	if err != nil {
		return err
	}

	// Initialize clients
	stacks, err := dockerCli.stacks()
	if err != nil {
		return err
	}
	composeClient, err := dockerCli.composeClient()
	if err != nil {
		return err
	}
	configMaps := composeClient.ConfigMaps()
	secrets := composeClient.Secrets()
	services := composeClient.Services()
	pods := composeClient.Pods()
	watcher := DeployWatcher{
		Pods: pods,
	}

	if err := stacks.IsColliding(services, stack); err != nil {
		return err
	}

	if err := stack.createFileBasedConfigMaps(configMaps); err != nil {
		return err
	}

	if err := stack.createFileBasedSecrets(secrets); err != nil {
		return err
	}

	if err = stacks.CreateOrUpdate(stack); err != nil {
		return err
	}

	fmt.Fprintln(cmdOut, "Waiting for the stack to be stable and running...")

	<-watcher.Watch(stack.name, stack.getServices())

	fmt.Fprintf(cmdOut, "Stack %s is stable and running\n\n", stack.name)

	return nil
}
