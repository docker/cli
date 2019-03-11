package stack

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/cli/opts"
	"github.com/docker/stacks/pkg/types"
)

// loadEnvFiles scans through the services, and replace any env_file blocks with their
// content
func loadEnvFiles(input *types.StackCreate, workingDir string) error {
	env, err := loadDotEnv(workingDir)
	if err != nil {
		return err
	}
	for i, service := range input.Spec.Services {
		// Override precedence
		// * Compose file
		// * Shell environment variables
		// * Environment file

		envFileSettings := map[string]*string{}
		if len(service.EnvFile) > 0 {
			var envVars []string
			for _, file := range service.EnvFile {
				filePath := absPath(workingDir, file)
				fileVars, err := opts.ParseEnvFile(filePath)
				if err != nil {
					return err
				}
				envVars = append(envVars, fileVars...)
			}
			envFileSettings = opts.ConvertKVStringsToMapWithNil(envVars)
		}

		newVars := map[string]*string{}
		for k, v := range service.Environment {
			if v == nil || *v == "" {
				envVal, envOK := env[k]
				fileVal, fileOK := envFileSettings[k]
				if envOK {
					newVars[k] = &envVal
					continue
				}
				if fileOK {
					newVars[k] = fileVal
					continue
				}
			}
			newVars[k] = v
		}
		// Append any missing file vars
		for k, v := range envFileSettings {
			if _, ok := newVars[k]; !ok {
				newVars[k] = v
			}
		}
		input.Spec.Services[i].Environment = newVars
	}
	return nil
}

func loadDotEnv(workingDir string) (map[string]string, error) {
	env := map[string]string{}
	fileOverrides, err := opts.ParseEnvFile(absPath(workingDir, ".env"))
	if err == nil {
		for _, line := range fileOverrides {
			e := strings.SplitN(line, "=", 2)
			if len(e) != 2 {
				return nil, fmt.Errorf("malformed env file %s - %s", absPath(workingDir, ".env"), line)
			}
			env[e[0]] = e[1]
		}
		// Shell takes precidence over the .env file
		for _, line := range os.Environ() {
			e := strings.SplitN(line, "=", 2)
			if len(e) != 2 {
				continue
			}
			env[e[0]] = e[1]
		}
	}
	return env, nil
}

func absPath(workingDir string, filePath string) string {
	if filepath.IsAbs(filePath) {
		return filePath
	}
	return filepath.Join(workingDir, filePath)
}
