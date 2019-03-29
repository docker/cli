package store

import (
	"testing"

	"gotest.tools/assert"
)

type testCtx struct{}
type testEP1 struct{}
type testEP2 struct{}
type testEP3 struct{}

func TestConfigModification(t *testing.T) {
	cfg := NewConfig(func() interface{} { return &testCtx{} }, EndpointTypeGetter("ep1", func() interface{} { return &testEP1{} }))
	assert.Equal(t, &testCtx{}, cfg.contextType())
	assert.Equal(t, &testEP1{}, cfg.endpointTypes["ep1"]())
	cfgCopy := cfg

	// modify existing endpoint
	cfg.SetEndpoint("ep1", func() interface{} { return &testEP2{} })
	// add endpoint
	cfg.SetEndpoint("ep2", func() interface{} { return &testEP3{} })
	assert.Equal(t, &testCtx{}, cfg.contextType())
	assert.Equal(t, &testEP2{}, cfg.endpointTypes["ep1"]())
	assert.Equal(t, &testEP3{}, cfg.endpointTypes["ep2"]())
	// check it applied on already initialized store
	assert.Equal(t, &testCtx{}, cfgCopy.contextType())
	assert.Equal(t, &testEP2{}, cfgCopy.endpointTypes["ep1"]())
	assert.Equal(t, &testEP3{}, cfgCopy.endpointTypes["ep2"]())
}
