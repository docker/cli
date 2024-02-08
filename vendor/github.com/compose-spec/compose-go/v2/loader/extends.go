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
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/compose-spec/compose-go/v2/consts"
	"github.com/compose-spec/compose-go/v2/override"
	"github.com/compose-spec/compose-go/v2/types"
)

func ApplyExtends(ctx context.Context, dict map[string]any, opts *Options, tracker *cycleTracker, post ...PostProcessor) error {
	a, ok := dict["services"]
	if !ok {
		return nil
	}
	services, ok := a.(map[string]any)
	if !ok {
		return fmt.Errorf("services must be a mapping")
	}
	for name := range services {
		merged, err := applyServiceExtends(ctx, name, services, opts, tracker, post...)
		if err != nil {
			return err
		}
		services[name] = merged
	}
	dict["services"] = services
	return nil
}

func applyServiceExtends(ctx context.Context, name string, services map[string]any, opts *Options, tracker *cycleTracker, post ...PostProcessor) (any, error) {
	s := services[name]
	if s == nil {
		return nil, nil
	}
	service, ok := s.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("services.%s must be a mapping", name)
	}
	extends, ok := service["extends"]
	if !ok {
		return s, nil
	}
	filename := ctx.Value(consts.ComposeFileKey{}).(string)
	var (
		err  error
		ref  string
		file any
	)
	switch v := extends.(type) {
	case map[string]any:
		ref = v["service"].(string)
		file = v["file"]
		opts.ProcessEvent("extends", v)
	case string:
		ref = v
		opts.ProcessEvent("extends", map[string]any{"service": ref})
	}

	var base any
	if file != nil {
		filename = file.(string)
		services, err = getExtendsBaseFromFile(ctx, ref, filename, opts, tracker)
		if err != nil {
			return nil, err
		}
	} else {
		_, ok := services[ref]
		if !ok {
			return nil, fmt.Errorf("cannot extend service %q in %s: service not found", name, filename)
		}
	}

	tracker, err = tracker.Add(filename, name)
	if err != nil {
		return nil, err
	}

	// recursively apply `extends`
	base, err = applyServiceExtends(ctx, ref, services, opts, tracker, post...)
	if err != nil {
		return nil, err
	}

	if base == nil {
		return service, nil
	}
	source := deepClone(base).(map[string]any)

	err = validateExtendSource(source, ref)
	if err != nil {
		return nil, err
	}

	for _, processor := range post {
		processor.Apply(map[string]any{
			"services": map[string]any{
				name: source,
			},
		})
	}
	merged, err := override.ExtendService(source, service)
	if err != nil {
		return nil, err
	}
	delete(merged, "extends")
	services[name] = merged
	return merged, nil
}

// validateExtendSource check the source for `extends` doesn't refer to another container/service
func validateExtendSource(source map[string]any, ref string) error {
	forbidden := []string{"links", "volumes_from", "depends_on"}
	for _, key := range forbidden {
		if _, ok := source[key]; ok {
			return fmt.Errorf("service %q can't be used with `extends` as it declare `%s`", ref, key)
		}
	}

	sharedNamespace := []string{"network_mode", "ipc", "pid", "net", "cgroup", "userns_mode", "uts"}
	for _, key := range sharedNamespace {
		if v, ok := source[key]; ok {
			val := v.(string)
			if strings.HasPrefix(val, types.ContainerPrefix) {
				return fmt.Errorf("service %q can't be used with `extends` as it shares `%s` with another container", ref, key)
			}
			if strings.HasPrefix(val, types.ServicePrefix) {
				return fmt.Errorf("service %q can't be used with `extends` as it shares `%s` with another service", ref, key)
			}
		}
	}
	return nil
}

func getExtendsBaseFromFile(ctx context.Context, name string, path string, opts *Options, ct *cycleTracker) (map[string]any, error) {
	for _, loader := range opts.ResourceLoaders {
		if !loader.Accept(path) {
			continue
		}
		local, err := loader.Load(ctx, path)
		if err != nil {
			return nil, err
		}
		localdir := filepath.Dir(local)
		relworkingdir := loader.Dir(path)

		extendsOpts := opts.clone()
		// replace localResourceLoader with a new flavour, using extended file base path
		extendsOpts.ResourceLoaders = append(opts.RemoteResourceLoaders(), localResourceLoader{
			WorkingDir: localdir,
		})
		extendsOpts.ResolvePaths = true
		extendsOpts.SkipNormalization = true
		extendsOpts.SkipConsistencyCheck = true
		extendsOpts.SkipInclude = true
		extendsOpts.SkipExtends = true    // we manage extends recursively based on raw service definition
		extendsOpts.SkipValidation = true // we validate the merge result
		extendsOpts.SkipDefaultValues = true
		source, err := loadYamlModel(ctx, types.ConfigDetails{
			WorkingDir: relworkingdir,
			ConfigFiles: []types.ConfigFile{
				{Filename: local},
			},
		}, extendsOpts, ct, nil)
		if err != nil {
			return nil, err
		}
		services := source["services"].(map[string]any)
		_, ok := services[name]
		if !ok {
			return nil, fmt.Errorf("cannot extend service %q in %s: service not found", name, path)
		}
		return services, nil
	}
	return nil, fmt.Errorf("cannot read %s", path)
}

func deepClone(value any) any {
	switch v := value.(type) {
	case []any:
		cp := make([]any, len(v))
		for i, e := range v {
			cp[i] = deepClone(e)
		}
		return cp
	case map[string]any:
		cp := make(map[string]any, len(v))
		for k, e := range v {
			cp[k] = deepClone(e)
		}
		return cp
	default:
		return value
	}
}
