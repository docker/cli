/*
   Copyright 2020 The Compose Specification Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package loader

import (
	"fmt"
	"strings"

	"github.com/compose-spec/compose-go/v2/errdefs"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/sirupsen/logrus"
)

// Normalize compose project by moving deprecated attributes to their canonical position and injecting implicit defaults
func Normalize(project *types.Project) error {
	if project.Networks == nil {
		project.Networks = make(map[string]types.NetworkConfig)
	}

	// If not declared explicitly, Compose model involves an implicit "default" network
	if _, ok := project.Networks["default"]; !ok {
		project.Networks["default"] = types.NetworkConfig{}
	}

	for name, s := range project.Services {
		if len(s.Networks) == 0 && s.NetworkMode == "" {
			// Service without explicit network attachment are implicitly exposed on default network
			s.Networks = map[string]*types.ServiceNetworkConfig{"default": nil}
		}

		if s.PullPolicy == types.PullPolicyIfNotPresent {
			s.PullPolicy = types.PullPolicyMissing
		}

		fn := func(s string) (string, bool) {
			v, ok := project.Environment[s]
			return v, ok
		}

		if s.Build != nil {
			if s.Build.Context == "" {
				s.Build.Context = "."
			}
			if s.Build.Dockerfile == "" && s.Build.DockerfileInline == "" {
				s.Build.Dockerfile = "Dockerfile"
			}
			s.Build.Args = s.Build.Args.Resolve(fn)
		}
		s.Environment = s.Environment.Resolve(fn)

		for _, link := range s.Links {
			parts := strings.Split(link, ":")
			if len(parts) == 2 {
				link = parts[0]
			}
			s.DependsOn = setIfMissing(s.DependsOn, link, types.ServiceDependency{
				Condition: types.ServiceConditionStarted,
				Restart:   true,
				Required:  true,
			})
		}

		for _, namespace := range []string{s.NetworkMode, s.Ipc, s.Pid, s.Uts, s.Cgroup} {
			if strings.HasPrefix(namespace, types.ServicePrefix) {
				name := namespace[len(types.ServicePrefix):]
				s.DependsOn = setIfMissing(s.DependsOn, name, types.ServiceDependency{
					Condition: types.ServiceConditionStarted,
					Restart:   true,
					Required:  true,
				})
			}
		}

		for _, vol := range s.VolumesFrom {
			if !strings.HasPrefix(vol, types.ContainerPrefix) {
				spec := strings.Split(vol, ":")
				s.DependsOn = setIfMissing(s.DependsOn, spec[0], types.ServiceDependency{
					Condition: types.ServiceConditionStarted,
					Restart:   false,
					Required:  true,
				})
			}
		}

		err := relocateLogDriver(&s)
		if err != nil {
			return err
		}

		err = relocateLogOpt(&s)
		if err != nil {
			return err
		}

		err = relocateDockerfile(&s)
		if err != nil {
			return err
		}

		inferImplicitDependencies(&s)

		project.Services[name] = s
	}

	setNameFromKey(project)

	return nil
}

// IsServiceDependency check the relation set by ref refers to a service
func IsServiceDependency(ref string) (string, bool) {
	if strings.HasPrefix(
		ref,
		types.ServicePrefix,
	) {
		return ref[len(types.ServicePrefix):], true
	}
	return "", false
}

func inferImplicitDependencies(service *types.ServiceConfig) {
	var dependencies []string

	maybeReferences := []string{
		service.NetworkMode,
		service.Ipc,
		service.Pid,
		service.Uts,
		service.Cgroup,
	}
	for _, ref := range maybeReferences {
		if dep, ok := IsServiceDependency(ref); ok {
			dependencies = append(dependencies, dep)
		}
	}

	for _, vol := range service.VolumesFrom {
		spec := strings.Split(vol, ":")
		if len(spec) == 0 {
			continue
		}
		if spec[0] == "container" {
			continue
		}
		dependencies = append(dependencies, spec[0])
	}

	for _, link := range service.Links {
		dependencies = append(dependencies, strings.Split(link, ":")[0])
	}

	if len(dependencies) > 0 && service.DependsOn == nil {
		service.DependsOn = make(types.DependsOnConfig)
	}

	for _, d := range dependencies {
		if _, ok := service.DependsOn[d]; !ok {
			service.DependsOn[d] = types.ServiceDependency{
				Condition: types.ServiceConditionStarted,
				Required:  true,
			}
		}
	}
}

// setIfMissing adds a ServiceDependency for service if not already defined
func setIfMissing(d types.DependsOnConfig, service string, dep types.ServiceDependency) types.DependsOnConfig {
	if d == nil {
		d = types.DependsOnConfig{}
	}
	if _, ok := d[service]; !ok {
		d[service] = dep
	}
	return d
}

// Resources with no explicit name are actually named by their key in map
func setNameFromKey(project *types.Project) {
	for key, n := range project.Networks {
		if n.Name == "" {
			if n.External {
				n.Name = key
			} else {
				n.Name = fmt.Sprintf("%s_%s", project.Name, key)
			}
			project.Networks[key] = n
		}
	}

	for key, v := range project.Volumes {
		if v.Name == "" {
			if v.External {
				v.Name = key
			} else {
				v.Name = fmt.Sprintf("%s_%s", project.Name, key)
			}
			project.Volumes[key] = v
		}
	}

	for key, c := range project.Configs {
		if c.Name == "" {
			if c.External {
				c.Name = key
			} else {
				c.Name = fmt.Sprintf("%s_%s", project.Name, key)
			}
			project.Configs[key] = c
		}
	}

	for key, s := range project.Secrets {
		if s.Name == "" {
			if s.External {
				s.Name = key
			} else {
				s.Name = fmt.Sprintf("%s_%s", project.Name, key)
			}
			project.Secrets[key] = s
		}
	}
}

func relocateLogOpt(s *types.ServiceConfig) error {
	if len(s.LogOpt) != 0 {
		logrus.Warn("`log_opts` is deprecated. Use the `logging` element")
		if s.Logging == nil {
			s.Logging = &types.LoggingConfig{}
		}
		for k, v := range s.LogOpt {
			if _, ok := s.Logging.Options[k]; !ok {
				s.Logging.Options[k] = v
			} else {
				return fmt.Errorf("can't use both 'log_opt' (deprecated) and 'logging.options': %w", errdefs.ErrInvalid)
			}
		}
	}
	return nil
}

func relocateLogDriver(s *types.ServiceConfig) error {
	if s.LogDriver != "" {
		logrus.Warn("`log_driver` is deprecated. Use the `logging` element")
		if s.Logging == nil {
			s.Logging = &types.LoggingConfig{}
		}
		if s.Logging.Driver == "" {
			s.Logging.Driver = s.LogDriver
		} else {
			return fmt.Errorf("can't use both 'log_driver' (deprecated) and 'logging.driver': %w", errdefs.ErrInvalid)
		}
	}
	return nil
}

func relocateDockerfile(s *types.ServiceConfig) error {
	if s.Dockerfile != "" {
		logrus.Warn("`dockerfile` is deprecated. Use the `build` element")
		if s.Build == nil {
			s.Build = &types.BuildConfig{}
		}
		if s.Dockerfile == "" {
			s.Build.Dockerfile = s.Dockerfile
		} else {
			return fmt.Errorf("can't use both 'dockerfile' (deprecated) and 'build.dockerfile': %w", errdefs.ErrInvalid)
		}
	}
	return nil
}
