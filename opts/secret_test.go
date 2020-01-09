package opts

import (
	"os"
	"testing"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func TestSecretOptions(t *testing.T) {
	testCases := []struct {
		name       string
		input      string
		secretName string
		fileName   string
		uid        string
		gid        string
		fileMode   uint
	}{
		{
			name:       "Simple",
			input:      "app-secret",
			secretName: "app-secret",
			fileName:   "app-secret",
			uid:        "0",
			gid:        "0",
		},
		{
			name:       "Source",
			input:      "source=foo",
			secretName: "foo",
			fileName:   "foo",
		},
		{
			name:       "SourceTarget",
			input:      "source=foo,target=testing",
			secretName: "foo",
			fileName:   "testing",
		},
		{
			name:       "Shorthand",
			input:      "src=foo,target=testing",
			secretName: "foo",
			fileName:   "testing",
		},
		{
			name:       "CustomUidGid",
			input:      "source=foo,target=testing,uid=1000,gid=1001",
			secretName: "foo",
			fileName:   "testing",
			uid:        "1000",
			gid:        "1001",
		},
		{
			name:       "CustomMode",
			input:      "source=foo,target=testing,uid=1000,gid=1001,mode=0444",
			secretName: "foo",
			fileName:   "testing",
			uid:        "1000",
			gid:        "1001",
			fileMode:   0444,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var opt SecretOpt
			assert.NilError(t, opt.Set(tc.input))
			reqs := opt.Value()
			assert.Assert(t, is.Len(reqs, 1))
			req := reqs[0]
			assert.Check(t, is.Equal(tc.secretName, req.SecretName))
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
