package manager

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"gotest.tools/assert"
	"gotest.tools/assert/cmp"
)

type fakeCandidate struct {
	path              string
	exec              bool
	meta              string
	allowExperimental bool
}

func (c *fakeCandidate) Experimental() bool {
	return strings.Contains(c.path, "-experimental/")
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
	var (
		goodPluginName = NamePrefix + "goodplugin"

		builtinName  = NamePrefix + "builtin"
		builtinAlias = NamePrefix + "alias"

		goodMeta = `{"SchemaVersion": "0.1.0", "Vendor": "e2e-testing"}`

		badPrefixPath          = "/usr/local/libexec/cli-plugins/wobble"
		badNamePath            = "/usr/local/libexec/cli-plugins/docker-123456"
		goodPluginPath         = "/usr/local/libexec/cli-plugins/" + goodPluginName
		experimentalPluginPath = "/usr/local/libexec/cli-plugins-experimental/" + goodPluginName
	)

	fakeroot := &cobra.Command{Use: "docker"}
	fakeroot.AddCommand(&cobra.Command{
		Use: strings.TrimPrefix(builtinName, NamePrefix),
		Aliases: []string{
			strings.TrimPrefix(builtinAlias, NamePrefix),
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
		{name: "bad prefix", c: &fakeCandidate{path: badPrefixPath}, err: fmt.Sprintf("does not have %q prefix", NamePrefix)},
		{name: "bad path", c: &fakeCandidate{path: badNamePath}, invalid: "did not match"},
		{name: "builtin command", c: &fakeCandidate{path: builtinName}, invalid: `plugin "builtin" duplicates builtin command`},
		{name: "builtin alias", c: &fakeCandidate{path: builtinAlias}, invalid: `plugin "alias" duplicates an alias of builtin command "builtin"`},
		{name: "fetch failure", c: &fakeCandidate{path: goodPluginPath, exec: false}, invalid: fmt.Sprintf("failed to fetch metadata: faked a failure to exec %q", goodPluginPath)},
		{name: "metadata not json", c: &fakeCandidate{path: goodPluginPath, exec: true, meta: `xyzzy`}, invalid: "invalid character"},
		{name: "empty schemaversion", c: &fakeCandidate{path: goodPluginPath, exec: true, meta: `{}`}, invalid: `plugin SchemaVersion "" is not valid`},
		{name: "invalid schemaversion", c: &fakeCandidate{path: goodPluginPath, exec: true, meta: `{"SchemaVersion": "xyzzy"}`}, invalid: `plugin SchemaVersion "xyzzy" is not valid`},
		{name: "no vendor", c: &fakeCandidate{path: goodPluginPath, exec: true, meta: `{"SchemaVersion": "0.1.0"}`}, invalid: "plugin metadata does not define a vendor"},
		{name: "empty vendor", c: &fakeCandidate{path: goodPluginPath, exec: true, meta: `{"SchemaVersion": "0.1.0", "Vendor": ""}`}, invalid: "plugin metadata does not define a vendor"},
		{name: "experimental required", c: &fakeCandidate{path: experimentalPluginPath, exec: true, meta: goodMeta}, invalid: "requires experimental CLI"},
		// This one should work
		{name: "valid", c: &fakeCandidate{path: goodPluginPath, exec: true, meta: goodMeta}},
		{name: "valid on experimental CLI", c: &fakeCandidate{path: goodPluginPath, exec: true, meta: goodMeta, allowExperimental: true}},
		{name: "experimental on experimental CLI", c: &fakeCandidate{path: experimentalPluginPath, exec: true, meta: goodMeta, allowExperimental: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, err := newPlugin(tc.c, fakeroot, tc.c.allowExperimental)
			if tc.err != "" {
				assert.ErrorContains(t, err, tc.err)
			} else if tc.invalid != "" {
				assert.NilError(t, err)
				assert.Assert(t, cmp.ErrorType(p.Err, reflect.TypeOf(&pluginError{})))
				assert.ErrorContains(t, p.Err, tc.invalid)
			} else {
				assert.NilError(t, err)
				assert.Equal(t, NamePrefix+p.Name, goodPluginName)
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
