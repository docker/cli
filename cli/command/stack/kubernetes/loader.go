package kubernetes

import (
	"github.com/docker/cli/cli/compose/loader"
	composetypes "github.com/docker/cli/cli/compose/types"
	yaml "gopkg.in/yaml.v2"
)

func loadStack(name string, cfg composetypes.Config) (stack, error) {
	res, err := yaml.Marshal(cfg)
	if err != nil {
		return stack{}, err
	}
	return stack{
		name:        name,
		composeFile: string(res),
		config:      &cfg,
	}, nil
}

func loadStackData(composefile string) (*composetypes.Config, error) {
	parsed, err := loader.ParseYAML([]byte(composefile))
	if err != nil {
		return nil, err
	}
	return loader.Load(composetypes.ConfigDetails{
		ConfigFiles: []composetypes.ConfigFile{
			{
				Config: parsed,
			},
		},
	})
}
