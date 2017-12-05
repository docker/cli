package kubernetes

import (
	"fmt"
	"io/ioutil"
	"path"

	"github.com/docker/cli/cli/command/stack/options"
	composeTypes "github.com/docker/cli/cli/compose/types"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// RunDeploy is the kubernetes implementation of docker stack deploy
func RunDeploy(dockerCli *KubeCli, opts options.Deploy) error {
	cmdOut := dockerCli.Out()
	// Check arguments
	if opts.Composefile == "" {
		return errors.Errorf("Please specify a Compose file (with --compose-file).")
	}
	// Initialize clients
	stackInterface, err := dockerCli.stacks()
	if err != nil {
		return err
	}
	composeClient, err := dockerCli.composeClient()
	if err != nil {
		return err
	}
	configMapInterface := composeClient.ConfigMaps()
	secretInterface := composeClient.Secrets()
	serviceInterface := composeClient.Services()
	podInterface := composeClient.Pods()
	watcher := DeployWatcher{
		Pods: podInterface,
	}

	// Parse the compose file
	stack, cfg, err := LoadStack(opts.Namespace, opts.Composefile)
	if err != nil {
		return err
	}

	// FIXME(vdemeester) handle warnings server-side
	if err = IsColliding(serviceInterface, stack, cfg); err != nil {
		return err
	}

	if err = createFileBasedConfigMaps(stack.Name, cfg.Configs, configMapInterface); err != nil {
		return err
	}

	if err = createFileBasedSecrets(stack.Name, cfg.Secrets, secretInterface); err != nil {
		return err
	}

	if in, err := stackInterface.Get(stack.Name, metav1.GetOptions{}); err == nil {
		in.Spec = stack.Spec

		if _, err = stackInterface.Update(in); err != nil {
			return err
		}

		fmt.Printf("Stack %s was updated\n", stack.Name)
	} else {
		if _, err = stackInterface.Create(stack); err != nil {
			return err
		}

		fmt.Fprintf(cmdOut, "Stack %s was created\n", stack.Name)
	}

	fmt.Fprintln(cmdOut, "Waiting for the stack to be stable and running...")

	<-watcher.Watch(stack, serviceNames(cfg))

	fmt.Fprintf(cmdOut, "Stack %s is stable and running\n\n", stack.Name)
	// fmt.Fprintf(cmdOut, "Read the logs with:\n  $ %s stack logs %s\n", filepath.Base(os.Args[0]), stack.Name)

	return nil
}

// createFileBasedConfigMaps creates a Kubernetes ConfigMap for each Compose global file-based config.
func createFileBasedConfigMaps(stackName string, globalConfigs map[string]composeTypes.ConfigObjConfig, configMaps corev1.ConfigMapInterface) error {
	for name, config := range globalConfigs {
		if config.File == "" {
			continue
		}

		fileName := path.Base(config.File)
		content, err := ioutil.ReadFile(config.File)
		if err != nil {
			return err
		}

		configMap := toConfigMap(stackName, name, fileName, content)
		configMaps.Create(configMap)
	}

	return nil
}

func serviceNames(cfg *composeTypes.Config) []string {
	names := []string{}

	for _, service := range cfg.Services {
		names = append(names, service.Name)
	}

	return names
}

// createFileBasedSecrets creates a Kubernetes Secret for each Compose global file-based secret.
func createFileBasedSecrets(stackName string, globalSecrets map[string]composeTypes.SecretConfig, secrets corev1.SecretInterface) error {
	for name, secret := range globalSecrets {
		if secret.File == "" {
			continue
		}

		fileName := path.Base(secret.File)
		content, err := ioutil.ReadFile(secret.File)
		if err != nil {
			return err
		}

		secret := toSecret(stackName, name, fileName, content)
		secrets.Create(secret)
	}

	return nil
}
