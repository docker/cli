package registry

import (
	"testing"

	cerrdefs "github.com/containerd/errdefs"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestLoadInsecureRegistries(t *testing.T) {
	testCases := []struct {
		registries []string
		index      string
		err        string
	}{
		{
			registries: []string{"127.0.0.1"},
			index:      "127.0.0.1",
		},
		{
			registries: []string{"127.0.0.1:8080"},
			index:      "127.0.0.1:8080",
		},
		{
			registries: []string{"2001:db8::1"},
			index:      "2001:db8::1",
		},
		{
			registries: []string{"[2001:db8::1]:80"},
			index:      "[2001:db8::1]:80",
		},
		{
			registries: []string{"http://myregistry.example.com"},
			index:      "myregistry.example.com",
		},
		{
			registries: []string{"https://myregistry.example.com"},
			index:      "myregistry.example.com",
		},
		{
			registries: []string{"HTTP://myregistry.example.com"},
			index:      "myregistry.example.com",
		},
		{
			registries: []string{"svn://myregistry.example.com"},
			err:        "insecure registry svn://myregistry.example.com should not contain '://'",
		},
		{
			registries: []string{"-invalid-registry"},
			err:        "Cannot begin or end with a hyphen",
		},
		{
			registries: []string{`mytest-.com`},
			err:        `insecure registry mytest-.com is not valid: invalid host "mytest-.com"`,
		},
		{
			registries: []string{`1200:0000:AB00:1234:0000:2552:7777:1313:8080`},
			err:        `insecure registry 1200:0000:AB00:1234:0000:2552:7777:1313:8080 is not valid: invalid host "1200:0000:AB00:1234:0000:2552:7777:1313:8080"`,
		},
		{
			registries: []string{`myregistry.example.com:500000`},
			err:        `insecure registry myregistry.example.com:500000 is not valid: invalid port "500000"`,
		},
		{
			registries: []string{`"myregistry.example.com"`},
			err:        `insecure registry "myregistry.example.com" is not valid: invalid host "\"myregistry.example.com\""`,
		},
		{
			registries: []string{`"myregistry.example.com:5000"`},
			err:        `insecure registry "myregistry.example.com:5000" is not valid: invalid host "\"myregistry.example.com"`,
		},
	}
	for _, testCase := range testCases {
		config := &serviceConfig{}
		err := config.loadInsecureRegistries(testCase.registries)
		if testCase.err == "" {
			if err != nil {
				t.Fatalf("expect no error, got '%s'", err)
			}
			match := false
			for index := range config.IndexConfigs {
				if index == testCase.index {
					match = true
				}
			}
			if !match {
				t.Fatalf("expect index configs to contain '%s', got %+v", testCase.index, config.IndexConfigs)
			}
		} else {
			if err == nil {
				t.Fatalf("expect error '%s', got no error", testCase.err)
			}
			assert.ErrorContains(t, err, testCase.err)
			assert.Check(t, cerrdefs.IsInvalidArgument(err))
		}
	}
}

func TestValidateIndexName(t *testing.T) {
	valid := []struct {
		index  string
		expect string
	}{
		{
			index:  "index.docker.io",
			expect: "docker.io",
		},
		{
			index:  "example.com",
			expect: "example.com",
		},
		{
			index:  "127.0.0.1:8080",
			expect: "127.0.0.1:8080",
		},
		{
			index:  "mytest-1.com",
			expect: "mytest-1.com",
		},
		{
			index:  "mirror-1.example.com/v1/?q=foo",
			expect: "mirror-1.example.com/v1/?q=foo",
		},
	}

	for _, testCase := range valid {
		result, err := ValidateIndexName(testCase.index)
		if assert.Check(t, err) {
			assert.Check(t, is.Equal(testCase.expect, result))
		}
	}
}

func TestValidateIndexNameWithError(t *testing.T) {
	invalid := []struct {
		index string
		err   string
	}{
		{
			index: "docker.io-",
			err:   "invalid index name (docker.io-). Cannot begin or end with a hyphen",
		},
		{
			index: "-example.com",
			err:   "invalid index name (-example.com). Cannot begin or end with a hyphen",
		},
		{
			index: "mirror-1.example.com/v1/?q=foo-",
			err:   "invalid index name (mirror-1.example.com/v1/?q=foo-). Cannot begin or end with a hyphen",
		},
	}
	for _, testCase := range invalid {
		_, err := ValidateIndexName(testCase.index)
		assert.Check(t, is.Error(err, testCase.err))
		assert.Check(t, cerrdefs.IsInvalidArgument(err))
	}
}
