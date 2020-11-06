package opts

import (
	"os"
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestConfigOptions(t *testing.T) {
	testCases := []struct {
		name       string
		input      string
		configName string
		fileName   string
		uid        string
		gid        string
		fileMode   uint
	}{
		{
			name:       "Simple",
			input:      "app-config",
			configName: "app-config",
			fileName:   "app-config",
			uid:        "0",
			gid:        "0",
		},
		{
			name:       "Source",
			input:      "source=foo",
			configName: "foo",
			fileName:   "foo",
		},
		{
			name:       "SourceTarget",
			input:      "source=foo,target=testing",
			configName: "foo",
			fileName:   "testing",
		},
		{
			name:       "Shorthand",
			input:      "src=foo,target=testing",
			configName: "foo",
			fileName:   "testing",
		},
		{
			name:       "CustomUidGid",
			input:      "source=foo,target=testing,uid=1000,gid=1001",
			configName: "foo",
			fileName:   "testing",
			uid:        "1000",
			gid:        "1001",
		},
		{
			name:       "CustomMode",
			input:      "source=foo,target=testing,uid=1000,gid=1001,mode=0444",
			configName: "foo",
			fileName:   "testing",
			uid:        "1000",
			gid:        "1001",
			fileMode:   0444,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var opt ConfigOpt
			assert.NilError(t, opt.Set(tc.input))
			reqs := opt.Value()
			assert.Assert(t, is.Len(reqs, 1))
			req := reqs[0]
			assert.Check(t, is.Equal(tc.configName, req.ConfigName))
			assert.Check(t, is.Equal(tc.fileName, req.File.Name))
			if tc.uid != "" {
				assert.Check(t, is.Equal(tc.uid, req.File.UID))
			}
			if tc.gid != "" {
				assert.Check(t, is.Equal(tc.gid, req.File.GID))
			}
			if tc.fileMode != 0 {
				assert.Check(t, is.Equal(os.FileMode(tc.fileMode), req.File.Mode))
			}
		})
	}
}
