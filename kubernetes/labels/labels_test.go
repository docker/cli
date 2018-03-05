package labels

import (
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
)

func TestForService(t *testing.T) {
	labels := ForService("stack", "service")

	assert.Check(t, is.Len(labels, 3))
	assert.Check(t, is.Equal("stack", labels["com.docker.stack.namespace"]))
	assert.Check(t, is.Equal("service", labels["com.docker.service.name"]))
	assert.Check(t, is.Equal("stack-service", labels["com.docker.service.id"]))
}

func TestSelectorForStack(t *testing.T) {
	assert.Check(t, is.Equal("com.docker.stack.namespace=demostack", SelectorForStack("demostack")))
	assert.Check(t, is.Equal("com.docker.stack.namespace=stack,com.docker.service.name=service", SelectorForStack("stack", "service")))
	assert.Check(t, is.Equal("com.docker.stack.namespace=stack,com.docker.service.name in (service1,service2)", SelectorForStack("stack", "service1", "service2")))
}
