// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.22

package loader

import (
	"context"
	"reflect"

	composeLoader "github.com/compose-spec/compose-go/v2/loader"
	compose "github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/cli/cli/compose/types"
	"golang.org/x/exp/slices"
)

// Load reads a ConfigDetails and returns a fully loaded configuration
func Load(configDetails compose.ConfigDetails, opt ...func(*composeLoader.Options)) (*compose.Project, error) {
	opts := []func(*composeLoader.Options){}
	opts = append(opts, opt...)
	clusterVolumeExtension := func(options *composeLoader.Options) {
		options.KnownExtensions = map[string]any{
			"x-cluster-spec": types.ClusterVolumeSpec{},
		}
	}
	opts = append(opts, clusterVolumeExtension)
	project, err := composeLoader.LoadWithContext(context.Background(), configDetails, opts...)
	if err != nil {
		return nil, err
	}

	if err := validateForbidden(project.Services); err != nil {
		return nil, err
	}
	return project, nil
}

func validateForbidden(services compose.Services) error {
	forbidden := getProperties(services, types.ForbiddenProperties)
	if len(forbidden) > 0 {
		return &ForbiddenPropertiesError{Properties: forbidden}
	}
	return nil
}

// GetUnsupportedProperties returns the list of any unsupported properties that are
// used in the Compose files.
func GetUnsupportedProperties(services compose.Services) []string {
	unsupported := []string{}

	for _, service := range services {
		r := reflect.ValueOf(service)
		for property, value := range types.UnsupportedProperties {
			f := reflect.Indirect(r).FieldByName(property)
			if f.IsValid() && !f.IsZero() && !slices.Contains(unsupported, value) {
				unsupported = append(unsupported, value)
			}
		}
	}
	slices.Sort(unsupported)
	return unsupported
}

// GetDeprecatedProperties returns the list of any deprecated properties that
// are used in the compose files.
func GetDeprecatedProperties(services compose.Services) map[string]string {
	deprecated := map[string]string{}

	deprecatedProperties := getProperties(services, types.DeprecatedProperties)
	for key, value := range deprecatedProperties {
		deprecated[key] = value
	}

	return deprecated
}

func getProperties(services compose.Services, propertyMap map[string]types.Pair[string, string]) map[string]string {
	output := map[string]string{}

	for _, service := range services {
		r := reflect.ValueOf(service)
		for property, pair := range propertyMap {
			f := reflect.Indirect(r).FieldByName(property)
			if f.IsValid() && !f.IsZero() {
				output[pair.Key()] = pair.Value()
			}
		}
	}

	return output
}

// ForbiddenPropertiesError is returned when there are properties in the Compose
// file that are forbidden.
type ForbiddenPropertiesError struct {
	Properties map[string]string
}

func (e *ForbiddenPropertiesError) Error() string {
	return "Configuration contains forbidden properties"
}
