package labels

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestForService(t *testing.T) {
	labels := ForService("stack", "service")

	assert.Len(t, labels, 3)
	assert.Equal(t, "stack", labels["com.docker.stack.namespace"])
	assert.Equal(t, "service", labels["com.docker.service.name"])
	assert.Equal(t, "stack-service", labels["com.docker.service.id"])
}

func TestSelectorForStack(t *testing.T) {
	assert.Equal(t, "com.docker.stack.namespace=demostack", SelectorForStack("demostack"))
	assert.Equal(t, "com.docker.stack.namespace=stack,com.docker.service.name=service", SelectorForStack("stack", "service"))
	assert.Equal(t, "com.docker.stack.namespace=stack,com.docker.service.name in (service1,service2)", SelectorForStack("stack", "service1", "service2"))
}
