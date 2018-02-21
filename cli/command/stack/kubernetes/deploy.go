package kubernetes

import (
	"fmt"
	"io/ioutil"
	"path"

	"github.com/docker/cli/cli/command/stack/loader"
	"github.com/docker/cli/cli/command/stack/options"
	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
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
	stack, err := LoadStack(opts.Namespace, *cfg)
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

	// FIXME(vdemeester) handle warnings server-side
	if err = IsColliding(services, stack, cfg); err != nil {
		return err
	}

	if err = createFileBasedConfigMaps(stack.Name, cfg.Configs, configMaps); err != nil {
		return err
	}

	if err = createFileBasedSecrets(stack.Name, cfg.Secrets, secrets); err != nil {
		return err
	}

	if in, err := stacks.Get(stack.Name, metav1.GetOptions{}); err == nil {
		in.Spec = stack.Spec

		if _, err = stacks.Update(in); err != nil {
			return err
		}

		fmt.Printf("Stack %s was updated\n", stack.Name)
	} else {
		if _, err = stacks.Create(stack); err != nil {
			return err
		}

		fmt.Fprintf(cmdOut, "Stack %s was created\n", stack.Name)
	}

	fmt.Fprintln(cmdOut, "Waiting for the stack to be stable and running...")

	<-watcher.Watch(stack, serviceNames(cfg))

	fmt.Fprintf(cmdOut, "Stack %s is stable and running\n\n", stack.Name)
	// TODO: fmt.Fprintf(cmdOut, "Read the logs with:\n  $ %s stack logs %s\n", filepath.Base(os.Args[0]), stack.Name)

	return nil
}

// createFileBasedConfigMaps creates a Kubernetes ConfigMap for each Compose global file-based config.
func createFileBasedConfigMaps(stackName string, globalConfigs map[string]composetypes.ConfigObjConfig, configMaps corev1.ConfigMapInterface) error {
	for name, config := range globalConfigs {
		if config.File == "" {
			continue
		}

		fileName := path.Base(config.File)
		content, err := ioutil.ReadFile(config.File)
		if err != nil {
			return err
		}

		if _, err := configMaps.Create(toConfigMap(stackName, name, fileName, content)); err != nil {
			return err
		}
	}

	return nil
}

func serviceNames(cfg *composetypes.Config) []string {
	names := []string{}

	for _, service := range cfg.Services {
		names = append(names, service.Name)
	}

	return names
}

// createFileBasedSecrets creates a Kubernetes Secret for each Compose global file-based secret.
func createFileBasedSecrets(stackName string, globalSecrets map[string]composetypes.SecretConfig, secrets corev1.SecretInterface) error {
	for name, secret := range globalSecrets {
		if secret.File == "" {
			continue
		}

		fileName := path.Base(secret.File)
		content, err := ioutil.ReadFile(secret.File)
		if err != nil {
			return err
		}

		if _, err := secrets.Create(toSecret(stackName, name, fileName, content)); err != nil {
			return err
		}
	}

	return nil
}
