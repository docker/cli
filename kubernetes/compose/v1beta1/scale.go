package v1beta1 // import "github.com/docker/cli/kubernetes/compose/v1beta1"

import (
	"errors"

	yaml "gopkg.in/yaml.v2"
)

// Scale scale a service to the specified number of replicas
func (s *Stack) Scale(service string, replicas int) (*Stack, error) {
	stack, err := s.Clone()
	if err != nil {
		return nil, err
	}
	var parsed yaml.MapSlice
	yaml.Unmarshal([]byte(stack.Spec.ComposeFile), &parsed)

	out, err := replace(parsed, "services", false, func(input yaml.MapSlice) (yaml.MapSlice, error) {
		return replace(input, service, false, func(input yaml.MapSlice) (yaml.MapSlice, error) {
			return replace(input, "deploy", true, func(input yaml.MapSlice) (yaml.MapSlice, error) {
				return replace(input, "replicas", true, replicas)
			})
		})
	})
	if err != nil {
		return nil, err
	}
	bin, err := yaml.Marshal(out)
	if err != nil {
		return nil, err
	}
	stack.Spec.ComposeFile = string(bin)
	return stack, nil
}

func replace(input yaml.MapSlice, key string, addIfNotFound bool, value interface{}) (yaml.MapSlice, error) {
	out := yaml.MapSlice{}
	found := false
	for _, i := range input {
		if i.Key == key {
			val, err := eval(value, i.Value)
			if err != nil {
				return nil, err
			}
			out = append(out, yaml.MapItem{
				Key:   i.Key,
				Value: val,
			})
			found = true
		} else {
			out = append(out, i)
		}
	}
	if !found {
		if !addIfNotFound {
			return nil, errors.New(key + " not found")
		}
		val, err := eval(value, yaml.MapSlice{})
		if err != nil {
			return nil, err
		}
		out = append(out, yaml.MapItem{
			Key:   key,
			Value: val,
		})
	}
	return out, nil
}

func eval(valueOrFun interface{}, input interface{}) (interface{}, error) {
	switch t := valueOrFun.(type) {
	case func(yaml.MapSlice) (yaml.MapSlice, error):
		return t(input.(yaml.MapSlice))
	default:
		return valueOrFun, nil
	}
}
