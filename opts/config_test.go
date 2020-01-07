package opts

import (
	"os"
	"testing"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func TestConfigOptionsSimple(t *testing.T) {
	var opt ConfigOpt

	testCase := "app-config"
	assert.NilError(t, opt.Set(testCase))

	reqs := opt.Value()
	assert.Assert(t, is.Len(reqs, 1))
	req := reqs[0]
	assert.Check(t, is.Equal("app-config", req.ConfigName))
	assert.Check(t, is.Equal("app-config", req.File.Name))
	assert.Check(t, is.Equal("0", req.File.UID))
	assert.Check(t, is.Equal("0", req.File.GID))
}

func TestConfigOptionsSource(t *testing.T) {
	var opt ConfigOpt

	testCase := "source=foo"
	assert.NilError(t, opt.Set(testCase))

	reqs := opt.Value()
	assert.Assert(t, is.Len(reqs, 1))
	req := reqs[0]
	assert.Check(t, is.Equal("foo", req.ConfigName))
}

func TestConfigOptionsSourceTarget(t *testing.T) {
	var opt ConfigOpt

	testCase := "source=foo,target=testing"
	assert.NilError(t, opt.Set(testCase))

	reqs := opt.Value()
	assert.Assert(t, is.Len(reqs, 1))
	req := reqs[0]
	assert.Check(t, is.Equal("foo", req.ConfigName))
	assert.Check(t, is.Equal("testing", req.File.Name))
}

func TestConfigOptionsShorthand(t *testing.T) {
	var opt ConfigOpt

	testCase := "src=foo,target=testing"
	assert.NilError(t, opt.Set(testCase))

	reqs := opt.Value()
	assert.Assert(t, is.Len(reqs, 1))
	req := reqs[0]
	assert.Check(t, is.Equal("foo", req.ConfigName))
}

func TestConfigOptionsCustomUidGid(t *testing.T) {
	var opt ConfigOpt

	testCase := "source=foo,target=testing,uid=1000,gid=1001"
	assert.NilError(t, opt.Set(testCase))

	reqs := opt.Value()
	assert.Assert(t, is.Len(reqs, 1))
	req := reqs[0]
	assert.Check(t, is.Equal("foo", req.ConfigName))
	assert.Check(t, is.Equal("testing", req.File.Name))
	assert.Check(t, is.Equal("1000", req.File.UID))
	assert.Check(t, is.Equal("1001", req.File.GID))
}

func TestConfigOptionsCustomMode(t *testing.T) {
	var opt ConfigOpt

	testCase := "source=foo,target=testing,uid=1000,gid=1001,mode=0444"
	assert.NilError(t, opt.Set(testCase))

	reqs := opt.Value()
	assert.Assert(t, is.Len(reqs, 1))
	req := reqs[0]
	assert.Check(t, is.Equal("foo", req.ConfigName))
	assert.Check(t, is.Equal("testing", req.File.Name))
	assert.Check(t, is.Equal("1000", req.File.UID))
	assert.Check(t, is.Equal("1001", req.File.GID))
	assert.Check(t, is.Equal(os.FileMode(0444), req.File.Mode))
}
