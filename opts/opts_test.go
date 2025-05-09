package opts

import (
	"fmt"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestValidateIPAddress(t *testing.T) {
	tests := []struct {
		doc         string
		input       string
		expectedOut string
		expectedErr string
	}{
		{
			doc:         "IPv4 loopback",
			input:       `127.0.0.1`,
			expectedOut: `127.0.0.1`,
		},
		{
			doc:         "IPv4 loopback with whitespace",
			input:       ` 127.0.0.1 `,
			expectedOut: `127.0.0.1`,
		},
		{
			doc:         "IPv6 loopback long form",
			input:       `0:0:0:0:0:0:0:1`,
			expectedOut: `::1`,
		},
		{
			doc:         "IPv6 loopback",
			input:       `::1`,
			expectedOut: `::1`,
		},
		{
			doc:         "IPv6 loopback with whitespace",
			input:       ` ::1 `,
			expectedOut: `::1`,
		},
		{
			doc:         "IPv6 lowercase",
			input:       `2001:db8::68`,
			expectedOut: `2001:db8::68`,
		},
		{
			doc:         "IPv6 uppercase",
			input:       `2001:DB8::68`,
			expectedOut: `2001:db8::68`,
		},
		{
			doc:         "IPv6 with brackets",
			input:       `[::1]`,
			expectedErr: `IP address is not correctly formatted: [::1]`,
		},
		{
			doc:         "IPv4 partial",
			input:       `127`,
			expectedErr: `IP address is not correctly formatted: 127`,
		},
		{
			doc:         "random invalid string",
			input:       `random invalid string`,
			expectedErr: `IP address is not correctly formatted: random invalid string`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			actualOut, actualErr := ValidateIPAddress(tc.input)
			assert.Check(t, is.Equal(tc.expectedOut, actualOut))
			if tc.expectedErr == "" {
				assert.Check(t, actualErr)
			} else {
				assert.Check(t, is.Error(actualErr, tc.expectedErr))
			}
		})
	}
}

func TestMapOpts(t *testing.T) {
	tmpMap := make(map[string]string)
	o := NewMapOpts(tmpMap, sampleValidator)
	err := o.Set("valid-option=1")
	if err != nil {
		t.Error(err)
	}
	if o.String() != "map[valid-option:1]" {
		t.Errorf("%s != [map[valid-option:1]", o.String())
	}

	err = o.Set("valid-option2=2")
	if err != nil {
		t.Error(err)
	}
	if len(tmpMap) != 2 {
		t.Errorf("map length %d != 2", len(tmpMap))
	}

	if tmpMap["valid-option"] != "1" {
		t.Errorf("valid-option = %s != 1", tmpMap["valid-option"])
	}
	if tmpMap["valid-option2"] != "2" {
		t.Errorf("valid-option2 = %s != 2", tmpMap["valid-option2"])
	}

	if o.Set("dummy-val=3") == nil {
		t.Error("validator is not being called")
	}
}

//nolint:gocyclo // ignore "cyclomatic complexity 17 is too high"
func TestListOptsWithoutValidator(t *testing.T) {
	o := NewListOpts(nil)
	err := o.Set("foo")
	if err != nil {
		t.Error(err)
	}
	if o.String() != "[foo]" {
		t.Errorf("%s != [foo]", o.String())
	}
	err = o.Set("bar")
	if err != nil {
		t.Error(err)
	}
	if o.Len() != 2 {
		t.Errorf("%d != 2", o.Len())
	}
	err = o.Set("bar")
	if err != nil {
		t.Error(err)
	}
	if o.Len() != 3 {
		t.Errorf("%d != 3", o.Len())
	}
	if !o.Get("bar") {
		t.Error(`o.Get("bar") == false`)
	}
	if o.Get("baz") {
		t.Error(`o.Get("baz") == true`)
	}
	o.Delete("foo")
	if o.String() != "[bar bar]" {
		t.Errorf("%s != [bar bar]", o.String())
	}
	if listOpts := o.GetAll(); len(listOpts) != 2 || listOpts[0] != "bar" || listOpts[1] != "bar" {
		t.Errorf("Expected [[bar bar]], got [%v]", listOpts)
	}
	if listOpts := o.GetSlice(); len(listOpts) != 2 || listOpts[0] != "bar" || listOpts[1] != "bar" {
		t.Errorf("Expected [[bar bar]], got [%v]", listOpts)
	}
	if mapListOpts := o.GetMap(); len(mapListOpts) != 1 {
		t.Errorf("Expected [map[bar:{}]], got [%v]", mapListOpts)
	}
}

func TestListOptsWithValidator(t *testing.T) {
	o := NewListOpts(sampleValidator)
	err := o.Set("foo")
	if err == nil {
		t.Error(err)
	}
	if o.String() != "" {
		t.Errorf(`%s != ""`, o.String())
	}
	err = o.Set("foo=bar")
	if err == nil {
		t.Error(err)
	}
	if o.String() != "" {
		t.Errorf(`%s != ""`, o.String())
	}
	err = o.Set("valid-option2=2")
	if err != nil {
		t.Error(err)
	}
	if o.Len() != 1 {
		t.Errorf("%d != 1", o.Len())
	}
	if !o.Get("valid-option2=2") {
		t.Error(`o.Get("valid-option2=2") == false`)
	}
	if o.Get("baz") {
		t.Error(`o.Get("baz") == true`)
	}
	o.Delete("valid-option2=2")
	if o.String() != "" {
		t.Errorf(`%s != ""`, o.String())
	}
}

func TestValidateDNSSearch(t *testing.T) {
	valid := []string{
		`.`,
		`a`,
		`a.`,
		`1.foo`,
		`17.foo`,
		`foo.bar`,
		`foo.bar.baz`,
		`foo.bar.`,
		`foo.bar.baz`,
		`foo1.bar2`,
		`foo1.bar2.baz`,
		`1foo.2bar.`,
		`1foo.2bar.baz`,
		`foo-1.bar-2`,
		`foo-1.bar-2.baz`,
		`foo-1.bar-2.`,
		`foo-1.bar-2.baz`,
		`1-foo.2-bar`,
		`1-foo.2-bar.baz`,
		`1-foo.2-bar.`,
		`1-foo.2-bar.baz`,
	}

	invalid := []string{
		``,
		` `,
		`  `,
		`17`,
		`17.`,
		`.17`,
		`17-.`,
		`17-.foo`,
		`.foo`,
		`foo-.bar`,
		`-foo.bar`,
		`foo.bar-`,
		`foo.bar-.baz`,
		`foo.-bar`,
		`foo.-bar.baz`,
		`foo.bar.baz.` +
			`this.should.fail.on.long.name.because.it.is.longer.thanisshouldbe` +
			`this.should.fail.on.long.name.because.it.is.longer.thanisshouldbe` +
			`this.should.fail.on.long.name.because.it.is.longer.thanisshouldbe` +
			`this.should.fail.on.long.name.because.it.is.longer.thanisshouldbe`,
	}

	for _, domain := range valid {
		if ret, err := ValidateDNSSearch(domain); err != nil || ret == "" {
			t.Fatalf("ValidateDNSSearch(`"+domain+"`) got %s %s", ret, err)
		}
	}

	for _, domain := range invalid {
		if ret, err := ValidateDNSSearch(domain); err == nil || ret != "" {
			t.Fatalf("ValidateDNSSearch(`"+domain+"`) got %s %s", ret, err)
		}
	}
}

func TestValidateLabel(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		expectedErr string
	}{
		{
			name:        "empty",
			expectedErr: `invalid label '': empty name`,
		},
		{
			name:        "whitespace only ",
			value:       " ",
			expectedErr: `invalid label ' ': empty name`,
		},
		{
			name:        "whitespace around equal-sign",
			value:       " = ",
			expectedErr: `invalid label ' = ': empty name`,
		},
		{
			name:  "leading whitespace",
			value: "    label=value",
		},
		{
			name:        "whitespaces in key without value",
			value:       "this is a label without value",
			expectedErr: `label 'this is a label without value' contains whitespaces`,
		},
		{
			name:        "whitespaces in key",
			value:       "this is a label=value",
			expectedErr: `label 'this is a label' contains whitespaces`,
		},
		{
			name:  "whitespaces in value",
			value: "label=a value that has whitespace",
		},
		{
			name:  "trailing whitespace in value",
			value: "label=value      ",
		},
		{
			name:  "leading whitespace in value",
			value: "label=    value",
		},
		{
			name:  "no value",
			value: "label",
		},
		{
			name:        "no key",
			value:       "=label",
			expectedErr: `invalid label '=label': empty name`,
		},
		{
			name:  "empty value",
			value: "label=",
		},
		{
			name:  "key value",
			value: "key1=value1",
		},
		{
			name:  "double equal-signs",
			value: "key1=value1=value2",
		},
		{
			name:  "multiple equal-signs",
			value: "key1=value1=value2=value",
		},
		{
			name:  "double quotes in key and value",
			value: `key"with"quotes={"hello"}`,
		},
		{
			name:  "double quotes around key and value",
			value: `"quoted-label"="quoted value"`,
		},
		{
			name:  "single quotes in key and value",
			value: `key'with'quotes=hello'with'quotes`,
		},
		{
			name:  "single quotes around key and value",
			value: `'quoted-label'='quoted value''`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			val, err := ValidateLabel(tc.value)
			if tc.expectedErr != "" {
				assert.Error(t, err, tc.expectedErr)
				return
			}
			assert.NilError(t, err)
			assert.Equal(t, val, tc.value)
		})
	}
}

func sampleValidator(val string) (string, error) {
	allowedKeys := map[string]string{"valid-option": "1", "valid-option2": "2"}
	k, _, _ := strings.Cut(val, "=")
	if allowedKeys[k] != "" {
		return val, nil
	}
	return "", fmt.Errorf("invalid key %s", k)
}

func TestNamedListOpts(t *testing.T) {
	var v []string
	o := NewNamedListOptsRef("foo-name", &v, nil)

	o.Set("foo")
	if o.String() != "[foo]" {
		t.Errorf("%s != [foo]", o.String())
	}
	if o.Name() != "foo-name" {
		t.Errorf("%s != foo-name", o.Name())
	}
	if len(v) != 1 {
		t.Errorf("expected foo to be in the values, got %v", v)
	}
}

func TestNamedMapOpts(t *testing.T) {
	tmpMap := make(map[string]string)
	o := NewNamedMapOpts("max-name", tmpMap, nil)

	o.Set("max-size=1")
	if o.String() != "map[max-size:1]" {
		t.Errorf("%s != [map[max-size:1]", o.String())
	}
	if o.Name() != "max-name" {
		t.Errorf("%s != max-name", o.Name())
	}
	if _, exist := tmpMap["max-size"]; !exist {
		t.Errorf("expected map-size to be in the values, got %v", tmpMap)
	}
}

func TestValidateMACAddress(t *testing.T) {
	if _, err := ValidateMACAddress(`92:d0:c6:0a:29:33`); err != nil {
		t.Fatalf("ValidateMACAddress(`92:d0:c6:0a:29:33`) got %s", err)
	}

	if _, err := ValidateMACAddress(`92:d0:c6:0a:33`); err == nil {
		t.Fatalf("ValidateMACAddress(`92:d0:c6:0a:33`) succeeded; expected failure on invalid MAC")
	}

	if _, err := ValidateMACAddress(`random invalid string`); err == nil {
		t.Fatalf("ValidateMACAddress(`random invalid string`) succeeded; expected failure on invalid MAC")
	}
}

func TestValidateLink(t *testing.T) {
	valid := []string{
		"name",
		"dcdfbe62ecd0:alias",
		"7a67485460b7642516a4ad82ecefe7f57d0c4916f530561b71a50a3f9c4e33da",
		"angry_torvalds:linus",
	}
	invalid := map[string]string{
		"":               "empty string specified for links",
		"too:much:of:it": "bad format for links: too:much:of:it",
	}

	for _, link := range valid {
		if _, err := ValidateLink(link); err != nil {
			t.Fatalf("ValidateLink(`%q`) should succeed: error %q", link, err)
		}
	}

	for link, expectedError := range invalid {
		if _, err := ValidateLink(link); err == nil {
			t.Fatalf("ValidateLink(`%q`) should have failed validation", link)
		} else if !strings.Contains(err.Error(), expectedError) {
			t.Fatalf("ValidateLink(`%q`) error should contain %q", link, expectedError)
		}
	}
}

func TestParseLink(t *testing.T) {
	name, alias, err := ParseLink("name:alias")
	if err != nil {
		t.Fatalf("Expected not to error out on a valid name:alias format but got: %v", err)
	}
	if name != "name" {
		t.Fatalf("Link name should have been name, got %s instead", name)
	}
	if alias != "alias" {
		t.Fatalf("Link alias should have been alias, got %s instead", alias)
	}
	// short format definition
	name, alias, err = ParseLink("name")
	if err != nil {
		t.Fatalf("Expected not to error out on a valid name only format but got: %v", err)
	}
	if name != "name" {
		t.Fatalf("Link name should have been name, got %s instead", name)
	}
	if alias != "name" {
		t.Fatalf("Link alias should have been name, got %s instead", alias)
	}
	// empty string link definition is not allowed
	if _, _, err := ParseLink(""); err == nil || !strings.Contains(err.Error(), "empty string specified for links") {
		t.Fatalf("Expected error 'empty string specified for links' but got: %v", err)
	}
	// more than two colons are not allowed
	if _, _, err := ParseLink("link:alias:wrong"); err == nil || !strings.Contains(err.Error(), "bad format for links: link:alias:wrong") {
		t.Fatalf("Expected error 'bad format for links: link:alias:wrong' but got: %v", err)
	}
}

func TestGetAllOrEmptyReturnsNilOrValue(t *testing.T) {
	opts := NewListOpts(nil)
	assert.Check(t, is.DeepEqual(opts.GetAllOrEmpty(), []string{}))
	opts.Set("foo")
	assert.Check(t, is.DeepEqual(opts.GetAllOrEmpty(), []string{"foo"}))
}

func TestParseCPUsReturnZeroOnInvalidValues(t *testing.T) {
	resValue, _ := ParseCPUs("foo")
	var z1 int64 = 0
	assert.Equal(t, z1, resValue)
	resValue, _ = ParseCPUs("1e-32")
	assert.Equal(t, z1, resValue)
}
