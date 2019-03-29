package loader

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/compose/loader"
	"github.com/docker/cli/cli/compose/schema"
	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/pkg/errors"
)

// LoadComposefile parse the composefile specified in the cli and returns its Config and version.
func LoadComposefile(dockerCli command.Cli, opts options.Deploy) (*composetypes.Config, error) {
	configDetails, err := getConfigDetails(opts.Composefiles, dockerCli.In())
	if err != nil {
		return nil, err
	}

	dicts := getDictsFrom(configDetails.ConfigFiles)
	config, err := loader.Load(configDetails)
	if err != nil {
		if fpe, ok := err.(*loader.ForbiddenPropertiesError); ok {
			return nil, errors.Errorf("Compose file contains unsupported options:\n\n%s\n",
				propertyWarnings(fpe.Properties))
		}

		return nil, err
	}

	unsupportedProperties := loader.GetUnsupportedProperties(dicts...)
	if len(unsupportedProperties) > 0 {
		fmt.Fprintf(dockerCli.Err(), "Ignoring unsupported options: %s\n\n",
			strings.Join(unsupportedProperties, ", "))
	}

	deprecatedProperties := loader.GetDeprecatedProperties(dicts...)
	if len(deprecatedProperties) > 0 {
		fmt.Fprintf(dockerCli.Err(), "Ignoring deprecated options:\n\n%s\n\n",
			propertyWarnings(deprecatedProperties))
	}
	return config, nil
}

func getDictsFrom(configFiles []composetypes.ConfigFile) []map[string]interface{} {
	dicts := []map[string]interface{}{}

	for _, configFile := range configFiles {
		dicts = append(dicts, configFile.Config)
	}

	return dicts
}

func propertyWarnings(properties map[string]string) string {
	var msgs []string
	for name, description := range properties {
		msgs = append(msgs, fmt.Sprintf("%s: %s", name, description))
	}
	sort.Strings(msgs)
	return strings.Join(msgs, "\n\n")
}

func getConfigDetails(composefiles []string, stdin io.Reader) (composetypes.ConfigDetails, error) {
	var details composetypes.ConfigDetails

	if len(composefiles) == 0 {
		return details, errors.New("no composefile(s)")
	}

	if composefiles[0] == "-" && len(composefiles) == 1 {
		workingDir, err := os.Getwd()
		if err != nil {
			return details, err
		}
		details.WorkingDir = workingDir
	} else {
		absPath, err := filepath.Abs(composefiles[0])
		if err != nil {
			return details, err
		}
		details.WorkingDir = filepath.Dir(absPath)
	}

	var err error
	details.ConfigFiles, err = loadConfigFiles(composefiles, stdin)
	if err != nil {
		return details, err
	}
	// Take the first file version (2 files can't have different version)
	details.Version = schema.Version(details.ConfigFiles[0].Config)
	details.Environment, err = buildEnvironment(os.Environ())
	return details, err
}

func buildEnvironment(env []string) (map[string]string, error) {
	result := make(map[string]string, len(env))
	for _, s := range env {
		// if value is empty, s is like "K=", not "K".
		if !strings.Contains(s, "=") {
			return result, errors.Errorf("unexpected environment %q", s)
		}
		kv := strings.SplitN(s, "=", 2)
		result[kv[0]] = kv[1]
	}
	return result, nil
}

func loadConfigFiles(filenames []string, stdin io.Reader) ([]composetypes.ConfigFile, error) {
	var configFiles []composetypes.ConfigFile

	for _, filename := range filenames {
		configFile, err := loadConfigFile(filename, stdin)
		if err != nil {
			return configFiles, err
		}
		configFiles = append(configFiles, *configFile)
	}

	return configFiles, nil
}

func loadConfigFile(filename string, stdin io.Reader) (*composetypes.ConfigFile, error) {
	var bytes []byte
	var err error

	if filename == "-" {
		bytes, err = ioutil.ReadAll(stdin)
	} else {
		bytes, err = ioutil.ReadFile(filename)
	}
	if err != nil {
		return nil, err
	}

	config, err := loader.ParseYAML(bytes)
	if err != nil {
		return nil, err
	}

	return &composetypes.ConfigFile{
		Filename: filename,
		Config:   config,
	}, nil
}
