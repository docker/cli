package labels

import (
	"fmt"
	"strings"
)

const (
	// ForServiceName is the label for the service name.
	ForServiceName = "com.docker.service.name"
	// ForStackName is the label for the stack name.
	ForStackName = "com.docker.stack.namespace"
	// ForServiceID is the label for the service id.
	ForServiceID = "com.docker.service.id"
)

// ForService gives the labels to select a given service in a stack.
func ForService(stackName, serviceName string) map[string]string {
	labels := map[string]string{}

	if serviceName != "" {
		labels[ForServiceName] = serviceName
	}
	if stackName != "" {
		labels[ForStackName] = stackName
	}
	if serviceName != "" && stackName != "" {
		labels[ForServiceID] = stackName + "-" + serviceName
	}

	return labels
}

// SelectorForStack gives the labelSelector to use for a given stack.
// Specific service names can be passed to narrow down the selection.
func SelectorForStack(stackName string, serviceNames ...string) string {
	switch len(serviceNames) {
	case 0:
		return fmt.Sprintf("%s=%s", ForStackName, stackName)
	case 1:
		return fmt.Sprintf("%s=%s,%s=%s", ForStackName, stackName, ForServiceName, serviceNames[0])
	default:
		return fmt.Sprintf("%s=%s,%s in (%s)", ForStackName, stackName, ForServiceName, strings.Join(serviceNames, ","))
	}
}
