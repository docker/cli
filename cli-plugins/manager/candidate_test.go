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

		builtinName  = metadata.NamePrefix + "builtin"
		builtinAlias = metadata.NamePrefix + "alias"

		badPrefixPath    = "/usr/local/libexec/cli-plugins/wobble"
		badNamePath      = "/usr/local/libexec/cli-plugins/docker-123456"
		goodPluginPath   = "/usr/local/libexec/cli-plugins/" + goodPluginName
		metaExperimental = `{"SchemaVersion": "0.1.0", "Vendor": "e2e-testing", "Experimental": true}`
	)

	fakeroot := &cobra.Command{Use: "docker"}
	fakeroot.AddCommand(&cobra.Command{
		Use: strings.TrimPrefix(builtinName, metadata.NamePrefix),
		Aliases: []string{
			strings.TrimPrefix(builtinAlias, metadata.NamePrefix),
		},
	})

	for _, tc := range []struct {
		name string
		c    *fakeCandidate

		// Either err or invalid may be non-empty, but not both (both can be empty for a good plugin).
		err     string
		invalid string
	}{
		/* Each failing one of the tests */
		{name: "empty path", c: &fakeCandidate{path: ""}, err: "plugin candidate path cannot be empty"},
		{name: "bad prefix", c: &fakeCandidate{path: badPrefixPath}, err: fmt.Sprintf("does not have %q prefix", metadata.NamePrefix)},
		{name: "bad path", c: &fakeCandidate{path: badNamePath}, invalid: "did not match"},
		{name: "builtin command", c: &fakeCandidate{path: builtinName}, invalid: `plugin "builtin" duplicates builtin command`},
		{name: "builtin alias", c: &fakeCandidate{path: builtinAlias}, invalid: `plugin "alias" duplicates an alias of builtin command "builtin"`},
		{name: "fetch failure", c: &fakeCandidate{path: goodPluginPath, exec: false}, invalid: fmt.Sprintf("failed to fetch metadata: faked a failure to exec %q", goodPluginPath)},
		{name: "metadata not json", c: &fakeCandidate{path: goodPluginPath, exec: true, meta: `xyzzy`}, invalid: "invalid character"},
		{name: "empty schemaversion", c: &fakeCandidate{path: goodPluginPath, exec: true, meta: `{}`}, invalid: `plugin SchemaVersion "" is not valid`},
		{name: "invalid schemaversion", c: &fakeCandidate{path: goodPluginPath, exec: true, meta: `{"SchemaVersion": "xyzzy"}`}, invalid: `plugin SchemaVersion "xyzzy" is not valid`},
		{name: "no vendor", c: &fakeCandidate{path: goodPluginPath, exec: true, meta: `{"SchemaVersion": "0.1.0"}`}, invalid: "plugin metadata does not define a vendor"},
		{name: "empty vendor", c: &fakeCandidate{path: goodPluginPath, exec: true, meta: `{"SchemaVersion": "0.1.0", "Vendor": ""}`}, invalid: "plugin metadata does not define a vendor"},
		// This one should work
		{name: "valid", c: &fakeCandidate{path: goodPluginPath, exec: true, meta: `{"SchemaVersion": "0.1.0", "Vendor": "e2e-testing"}`}},
		{name: "experimental + allowing experimental", c: &fakeCandidate{path: goodPluginPath, exec: true, meta: metaExperimental}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, err := newPlugin(tc.c, fakeroot.Commands())
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
				assert.Equal(t, p.SchemaVersion, "0.1.0")
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
