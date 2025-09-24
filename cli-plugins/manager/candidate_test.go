package manager

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/docker/cli/cli-plugins/metadata"
	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

type fakeCandidate struct {
	path string
	exec bool
	meta string
}

func (c *fakeCandidate) Path() string {
	return c.path
}

func (c *fakeCandidate) Metadata() ([]byte, error) {
	if !c.exec {
		return nil, fmt.Errorf("faked a failure to exec %q", c.path)
	}
	return []byte(c.meta), nil
}

func TestValidateCandidate(t *testing.T) {
	const (
		goodPluginName = metadata.NamePrefix + "goodplugin"
		builtinName    = metadata.NamePrefix + "builtin"
		builtinAlias   = metadata.NamePrefix + "alias"

		badPrefixPath  = "/usr/local/libexec/cli-plugins/wobble"
		badNamePath    = "/usr/local/libexec/cli-plugins/docker-123456"
		goodPluginPath = "/usr/local/libexec/cli-plugins/" + goodPluginName
	)

	fakeroot := &cobra.Command{Use: "docker"}
	fakeroot.AddCommand(&cobra.Command{
		Use: strings.TrimPrefix(builtinName, metadata.NamePrefix),
		Aliases: []string{
			strings.TrimPrefix(builtinAlias, metadata.NamePrefix),
		},
	})

	for _, tc := range []struct {
		name   string
		plugin *fakeCandidate

		// Either err or invalid may be non-empty, but not both (both can be empty for a good plugin).
		err     string
		invalid string
		expVer  string
	}{
		// Invalid cases.
		{
			name:   "empty path",
			plugin: &fakeCandidate{path: ""},
			err:    "plugin candidate path cannot be empty",
		},
		{
			name:   "bad prefix",
			plugin: &fakeCandidate{path: badPrefixPath},
			err:    fmt.Sprintf("does not have %q prefix", metadata.NamePrefix),
		},
		{
			name:    "bad path",
			plugin:  &fakeCandidate{path: badNamePath},
			invalid: "did not match",
		},
		{
			name:    "builtin command",
			plugin:  &fakeCandidate{path: builtinName},
			invalid: `plugin "builtin" duplicates builtin command`,
		},
		{
			name:    "builtin alias",
			plugin:  &fakeCandidate{path: builtinAlias},
			invalid: `plugin "alias" duplicates an alias of builtin command "builtin"`,
		},
		{
			name:    "fetch failure",
			plugin:  &fakeCandidate{path: goodPluginPath, exec: false},
			invalid: fmt.Sprintf("failed to fetch metadata: faked a failure to exec %q", goodPluginPath),
		},
		{
			name:    "metadata not json",
			plugin:  &fakeCandidate{path: goodPluginPath, exec: true, meta: `xyzzy`},
			invalid: "invalid character",
		},
		{
			name:    "empty schemaversion",
			plugin:  &fakeCandidate{path: goodPluginPath, exec: true, meta: `{}`},
			invalid: `plugin SchemaVersion version cannot be empty`,
		},
		{
			name:    "invalid schemaversion",
			plugin:  &fakeCandidate{path: goodPluginPath, exec: true, meta: `{"SchemaVersion": "xyzzy"}`},
			invalid: `plugin SchemaVersion "xyzzy" has wrong format: must be <major>.<minor>.<patch>`,
		},
		{
			name:    "invalid schemaversion major",
			plugin:  &fakeCandidate{path: goodPluginPath, exec: true, meta: `{"SchemaVersion": "2.0.0"}`},
			invalid: `plugin SchemaVersion "2.0.0" is not supported: must be lower than 2.0.0`,
		},
		{
			name:    "no vendor",
			plugin:  &fakeCandidate{path: goodPluginPath, exec: true, meta: `{"SchemaVersion": "0.1.0"}`},
			invalid: "plugin metadata does not define a vendor",
		},
		{
			name:    "empty vendor",
			plugin:  &fakeCandidate{path: goodPluginPath, exec: true, meta: `{"SchemaVersion": "0.1.0", "Vendor": ""}`},
			invalid: "plugin metadata does not define a vendor",
		},

		// Valid cases.
		{
			name:   "valid",
			plugin: &fakeCandidate{path: goodPluginPath, exec: true, meta: `{"SchemaVersion": "0.1.0", "Vendor": "e2e-testing"}`},
			expVer: "0.1.0",
		},
		{
			// Including the deprecated "experimental" field should not break processing.
			name:   "with legacy experimental",
			plugin: &fakeCandidate{path: goodPluginPath, exec: true, meta: `{"SchemaVersion": "0.1.0", "Vendor": "e2e-testing", "Experimental": true}`},
			expVer: "0.1.0",
		},
		{
			// note that this may not be supported by older CLIs
			name:   "new minor schema version",
			plugin: &fakeCandidate{path: goodPluginPath, exec: true, meta: `{"SchemaVersion": "0.2.0", "Vendor": "e2e-testing"}`},
			expVer: "0.2.0",
		},
		{
			// note that this may not be supported by older CLIs
			name:   "new major schema version",
			plugin: &fakeCandidate{path: goodPluginPath, exec: true, meta: `{"SchemaVersion": "1.0.0", "Vendor": "e2e-testing"}`},
			expVer: "1.0.0",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, err := newPlugin(tc.plugin, fakeroot.Commands())
			switch {
			case tc.err != "":
				assert.ErrorContains(t, err, tc.err)
			case tc.invalid != "":
				assert.NilError(t, err)
				assert.Assert(t, is.ErrorType(p.Err, reflect.TypeOf(&pluginError{})))
				assert.ErrorContains(t, p.Err, tc.invalid)
			default:
				assert.NilError(t, err)
				assert.Equal(t, metadata.NamePrefix+p.Name, goodPluginName)
				assert.Equal(t, p.SchemaVersion, tc.expVer)
				assert.Equal(t, p.Vendor, "e2e-testing")
			}
		})
	}
}

func TestCandidatePath(t *testing.T) {
	exp := "/some/path"
	cand := &candidate{path: exp}
	assert.Equal(t, exp, cand.Path())
}
