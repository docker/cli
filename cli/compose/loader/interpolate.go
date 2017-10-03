package loader

import (
	"strconv"
	"strings"

	interp "github.com/docker/cli/cli/compose/interpolation"
	"github.com/pkg/errors"
)

var interpolateTypeCastMapping = map[string]map[interp.Path]interp.Cast{
	"services": {
		iPath("configs", "mode"):                              toInt,
		iPath("secrets", "mode"):                              toInt,
		iPath("healthcheck", "retries"):                       toInt,
		iPath("healthcheck", "disable"):                       toBoolean,
		iPath("deploy", "replicas"):                           toInt,
		iPath("deploy", "update_config", "parallelism:"):      toInt,
		iPath("deploy", "update_config", "max_failure_ratio"): toFloat,
		iPath("deploy", "restart_policy", "max_attempts"):     toInt,
		iPath("ports", "target"):                              toInt,
		iPath("ports", "published"):                           toInt,
		iPath("ulimits", interp.PathMatchAll):                 toInt,
		iPath("ulimits", interp.PathMatchAll, "hard"):         toInt,
		iPath("ulimits", interp.PathMatchAll, "soft"):         toInt,
		iPath("privileged"):                                   toBoolean,
		iPath("read_only"):                                    toBoolean,
		iPath("stdin_open"):                                   toBoolean,
		iPath("tty"):                                          toBoolean,
		iPath("volumes", "read_only"):                         toBoolean,
		iPath("volumes", "volume", "nocopy"):                  toBoolean,
	},
	"networks": {
		iPath("external"):   toBoolean,
		iPath("internal"):   toBoolean,
		iPath("attachable"): toBoolean,
	},
	"volumes": {
		iPath("external"): toBoolean,
	},
	"secrets": {
		iPath("external"): toBoolean,
	},
	"configs": {
		iPath("external"): toBoolean,
	},
}

func iPath(parts ...string) interp.Path {
	return interp.NewPath(append([]string{interp.PathMatchAll}, parts...)...)
}

func toInt(value string) (interface{}, error) {
	return strconv.Atoi(value)
}

func toFloat(value string) (interface{}, error) {
	return strconv.ParseFloat(value, 64)
}

// should match http://yaml.org/type/bool.html
func toBoolean(value string) (interface{}, error) {
	switch strings.ToLower(value) {
	case "y", "yes", "true", "on":
		return true, nil
	case "n", "no", "false", "off":
		return false, nil
	default:
		return nil, errors.Errorf("invalid boolean: %s", value)
	}
}

func interpolateConfig(configDict map[string]interface{}, lookupEnv interp.LookupValue) (map[string]map[string]interface{}, error) {
	config := make(map[string]map[string]interface{})

	for _, key := range []string{"services", "networks", "volumes", "secrets", "configs"} {
		section, ok := configDict[key]
		if !ok {
			config[key] = make(map[string]interface{})
			continue
		}
		var err error
		config[key], err = interp.Interpolate(
			section.(map[string]interface{}),
			interp.Options{
				SectionName:     key,
				LookupValue:     lookupEnv,
				TypeCastMapping: interpolateTypeCastMapping[key],
			})
		if err != nil {
			return nil, err
		}
	}
	return config, nil
}
