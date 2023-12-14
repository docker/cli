// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.19

package store

import (
	"testing"

	"gotest.tools/v3/assert"
)

type (
	testCtx struct{}
	testEP1 struct{}
	testEP2 struct{}
	testEP3 struct{}
)

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

func TestValidFilePaths(t *testing.T) {
	paths := map[string]bool{
		"tls/_/../../something":        false,
		"tls/../../something":          false,
		"../../something":              false,
		"/tls/absolute/unix/path":      false,
		`C:\tls\absolute\windows\path`: false,
		"C:/tls/absolute/windows/path": false,
	}
	for p, expectedValid := range paths {
		err := isValidFilePath(p)
		assert.Equal(t, err == nil, expectedValid, "%q should report valid as: %v", p, expectedValid)
	}
}

func TestValidateContextName(t *testing.T) {
	names := map[string]bool{
		"../../invalid/escape": false,
		"/invalid/absolute":    false,
		`\invalid\windows`:     false,
		"validname":            true,
	}
	for n, expectedValid := range names {
		err := ValidateContextName(n)
		assert.Equal(t, err == nil, expectedValid, "%q should report valid as: %v", n, expectedValid)
	}
}
