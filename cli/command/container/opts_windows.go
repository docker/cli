// +build windows

package container

import (
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/pkg/errors"
)

// parseDevice parses a device mapping string to a container.DeviceMapping struct
func parseDevice(device string) (container.DeviceMapping, error) {
	return container.DeviceMapping{
		PathOnHost: device,
	}, nil
}

// validateDevice validates a path for devices
// It will make sure 'val' is in the form:
//    class/{clsid}
func validateDevice(val string) (string, error) {
	arr := strings.Split(val, ":")
	switch len(arr) {
	case 1:
		if !strings.HasPrefix(arr[0], "class") {
			return "", errors.New("device must have prefix: 'class'")
		}
		return val, nil
	}

	return "", errors.Errorf("device must be in the format 'class/{clsid}'")
}
