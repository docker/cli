package kubernetes

import (
	"bufio"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/cli/cli/compose/loader"
	"github.com/docker/cli/cli/compose/template"
	composetypes "github.com/docker/cli/cli/compose/types"
	apiv1beta1 "github.com/docker/cli/kubernetes/compose/v1beta1"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LoadStack loads a stack from a Compose file, with a given name.
func LoadStack(name, composeFile string) (*apiv1beta1.Stack, *composetypes.Config, error) {
	if composeFile == "" {
		return nil, nil, errors.New("compose-file must be set")
	}

	workingDir, err := os.Getwd()
	if err != nil {
		return nil, nil, err
	}

	composePath := composeFile
	if !strings.HasPrefix(composePath, "/") {
		composePath = filepath.Join(workingDir, composeFile)
	}

	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		return nil, nil, errors.Errorf("no compose file found in %s", filepath.Dir(composePath))
	}

	binary, err := ioutil.ReadFile(composePath)
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot load compose file")
	}

	return load(name, binary, env())
}

func load(name string, binary []byte, env map[string]string) (*apiv1beta1.Stack, *composetypes.Config, error) {
	processed, err := template.Substitute(string(binary), func(key string) (string, bool) { return env[key], true })
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot load compose file")
	}

	parsed, err := loader.ParseYAML([]byte(processed))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "cannot load compose file")
	}

	cfg, err := loader.Load(composetypes.ConfigDetails{
		WorkingDir: ".",
		ConfigFiles: []composetypes.ConfigFile{
			{
				Config: parsed,
			},
		},
	})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "cannot load compose file")
	}

	result, err := processEnvFiles(processed, parsed, cfg)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "cannot load compose file")
	}

	return &apiv1beta1.Stack{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: apiv1beta1.StackSpec{
			ComposeFile: result,
		},
	}, cfg, nil
}

type iMap = map[string]interface{}

func processEnvFiles(input string, parsed map[string]interface{}, config *composetypes.Config) (string, error) {
	changed := false

	for _, svc := range config.Services {
		if len(svc.EnvFile) == 0 {
			continue
		}
		// Load() processed the env_file for us, we just need to inject back into
		// the intermediate representation
		env := iMap{}
		for k, v := range svc.Environment {
			env[k] = v
		}
		parsed["services"].(iMap)[svc.Name].(iMap)["environment"] = env
		delete(parsed["services"].(iMap)[svc.Name].(iMap), "env_file")
		changed = true
	}
	if !changed {
		return input, nil
	}
	res, err := yaml.Marshal(parsed)
	if err != nil {
		return "", err
	}
	return string(res), nil
}

func env() map[string]string {
	// Apply .env file first
	config := readEnvFile(".env")

	// Apply env variables
	for k, v := range envToMap(os.Environ()) {
		config[k] = v
	}

	return config
}

func readEnvFile(path string) map[string]string {
	config := map[string]string{}

	file, err := os.Open(path)
	if err != nil {
		return config // Ignore
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]

			config[key] = value
		}
	}

	return config
}

func envToMap(env []string) map[string]string {
	config := map[string]string{}

	for _, value := range env {
		parts := strings.SplitN(value, "=", 2)

		key := parts[0]
		value := parts[1]

		config[key] = value
	}

	return config
}
