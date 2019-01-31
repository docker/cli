package configfile

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/docker/cli/cli/config/credentials"
	"github.com/docker/cli/cli/config/types"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"gotest.tools/golden"
)

func TestEncodeAuth(t *testing.T) {
	newAuthConfig := &types.AuthConfig{Username: "ken", Password: "test"}
	authStr := encodeAuth(newAuthConfig)

	expected := &types.AuthConfig{}
	var err error
	expected.Username, expected.Password, err = decodeAuth(authStr)
	assert.NilError(t, err)
	assert.Check(t, is.DeepEqual(expected, newAuthConfig))
}

func TestProxyConfig(t *testing.T) {
	httpProxy := "http://proxy.mycorp.com:3128"
	httpsProxy := "https://user:password@proxy.mycorp.com:3129"
	ftpProxy := "http://ftpproxy.mycorp.com:21"
	noProxy := "*.intra.mycorp.com"
	defaultProxyConfig := ProxyConfig{
		HTTPProxy:  httpProxy,
		HTTPSProxy: httpsProxy,
		FTPProxy:   ftpProxy,
		NoProxy:    noProxy,
	}

	cfg := ConfigFile{
		Proxies: map[string]ProxyConfig{
			"default": defaultProxyConfig,
		},
	}

	proxyConfig := cfg.ParseProxyConfig("/var/run/docker.sock", nil)
	expected := map[string]*string{
		"HTTP_PROXY":  &httpProxy,
		"http_proxy":  &httpProxy,
		"HTTPS_PROXY": &httpsProxy,
		"https_proxy": &httpsProxy,
		"FTP_PROXY":   &ftpProxy,
		"ftp_proxy":   &ftpProxy,
		"NO_PROXY":    &noProxy,
		"no_proxy":    &noProxy,
	}
	assert.Check(t, is.DeepEqual(expected, proxyConfig))
}

func TestProxyConfigOverride(t *testing.T) {
	httpProxy := "http://proxy.mycorp.com:3128"
	overrideHTTPProxy := "http://proxy.example.com:3128"
	overrideNoProxy := ""
	httpsProxy := "https://user:password@proxy.mycorp.com:3129"
	ftpProxy := "http://ftpproxy.mycorp.com:21"
	noProxy := "*.intra.mycorp.com"
	defaultProxyConfig := ProxyConfig{
		HTTPProxy:  httpProxy,
		HTTPSProxy: httpsProxy,
		FTPProxy:   ftpProxy,
		NoProxy:    noProxy,
	}

	cfg := ConfigFile{
		Proxies: map[string]ProxyConfig{
			"default": defaultProxyConfig,
		},
	}

	clone := func(s string) *string {
		s2 := s
		return &s2
	}

	ropts := map[string]*string{
		"HTTP_PROXY": clone(overrideHTTPProxy),
		"NO_PROXY":   clone(overrideNoProxy),
	}
	proxyConfig := cfg.ParseProxyConfig("/var/run/docker.sock", ropts)
	expected := map[string]*string{
		"HTTP_PROXY":  &overrideHTTPProxy,
		"http_proxy":  &httpProxy,
		"HTTPS_PROXY": &httpsProxy,
		"https_proxy": &httpsProxy,
		"FTP_PROXY":   &ftpProxy,
		"ftp_proxy":   &ftpProxy,
		"NO_PROXY":    &overrideNoProxy,
		"no_proxy":    &noProxy,
	}
	assert.Check(t, is.DeepEqual(expected, proxyConfig))
}

func TestProxyConfigPerHost(t *testing.T) {
	httpProxy := "http://proxy.mycorp.com:3128"
	httpsProxy := "https://user:password@proxy.mycorp.com:3129"
	ftpProxy := "http://ftpproxy.mycorp.com:21"
	noProxy := "*.intra.mycorp.com"

	extHTTPProxy := "http://proxy.example.com:3128"
	extHTTPSProxy := "https://user:password@proxy.example.com:3129"
	extFTPProxy := "http://ftpproxy.example.com:21"
	extNoProxy := "*.intra.example.com"

	defaultProxyConfig := ProxyConfig{
		HTTPProxy:  httpProxy,
		HTTPSProxy: httpsProxy,
		FTPProxy:   ftpProxy,
		NoProxy:    noProxy,
	}
	externalProxyConfig := ProxyConfig{
		HTTPProxy:  extHTTPProxy,
		HTTPSProxy: extHTTPSProxy,
		FTPProxy:   extFTPProxy,
		NoProxy:    extNoProxy,
	}

	cfg := ConfigFile{
		Proxies: map[string]ProxyConfig{
			"default":                       defaultProxyConfig,
			"tcp://example.docker.com:2376": externalProxyConfig,
		},
	}

	proxyConfig := cfg.ParseProxyConfig("tcp://example.docker.com:2376", nil)
	expected := map[string]*string{
		"HTTP_PROXY":  &extHTTPProxy,
		"http_proxy":  &extHTTPProxy,
		"HTTPS_PROXY": &extHTTPSProxy,
		"https_proxy": &extHTTPSProxy,
		"FTP_PROXY":   &extFTPProxy,
		"ftp_proxy":   &extFTPProxy,
		"NO_PROXY":    &extNoProxy,
		"no_proxy":    &extNoProxy,
	}
	assert.Check(t, is.DeepEqual(expected, proxyConfig))
}

func TestConfigFile(t *testing.T) {
	configFilename := "configFilename"
	configFile := New(configFilename)

	assert.Check(t, is.Equal(configFilename, configFile.Filename))
}

type mockNativeStore struct {
	GetAllCallCount int
	authConfigs     map[string]types.AuthConfig
}

func (c *mockNativeStore) Erase(registryHostname string) error {
	delete(c.authConfigs, registryHostname)
	return nil
}

func (c *mockNativeStore) Get(registryHostname string) (types.AuthConfig, error) {
	return c.authConfigs[registryHostname], nil
}

func (c *mockNativeStore) GetAll() (map[string]types.AuthConfig, error) {
	c.GetAllCallCount = c.GetAllCallCount + 1
	return c.authConfigs, nil
}

func (c *mockNativeStore) Store(authConfig types.AuthConfig) error {
	return nil
}

// make sure it satisfies the interface
var _ credentials.Store = (*mockNativeStore)(nil)

func NewMockNativeStore(authConfigs map[string]types.AuthConfig) credentials.Store {
	return &mockNativeStore{authConfigs: authConfigs}
}

func TestGetAllCredentialsFileStoreOnly(t *testing.T) {
	configFile := New("filename")
	exampleAuth := types.AuthConfig{
		Username: "user",
		Password: "pass",
	}
	configFile.AuthConfigs["example.com/foo"] = exampleAuth

	authConfigs, err := configFile.GetAllCredentials()
	assert.NilError(t, err)

	expected := make(map[string]types.AuthConfig)
	expected["example.com/foo"] = exampleAuth
	assert.Check(t, is.DeepEqual(expected, authConfigs))
}

func TestGetAllCredentialsCredsStore(t *testing.T) {
	configFile := New("filename")
	configFile.CredentialsStore = "test_creds_store"
	testRegistryHostname := "example.com"
	expectedAuth := types.AuthConfig{
		Username: "user",
		Password: "pass",
	}

	testCredsStore := NewMockNativeStore(map[string]types.AuthConfig{testRegistryHostname: expectedAuth})

	tmpNewNativeStore := newNativeStore
	defer func() { newNativeStore = tmpNewNativeStore }()
	newNativeStore = func(configFile *ConfigFile, helperSuffix string) credentials.Store {
		return testCredsStore
	}

	authConfigs, err := configFile.GetAllCredentials()
	assert.NilError(t, err)

	expected := make(map[string]types.AuthConfig)
	expected[testRegistryHostname] = expectedAuth
	assert.Check(t, is.DeepEqual(expected, authConfigs))
	assert.Check(t, is.Equal(1, testCredsStore.(*mockNativeStore).GetAllCallCount))
}

func TestGetAllCredentialsCredHelper(t *testing.T) {
	testCredHelperSuffix := "test_cred_helper"
	testCredHelperRegistryHostname := "credhelper.com"
	testExtraCredHelperRegistryHostname := "somethingweird.com"

	unexpectedCredHelperAuth := types.AuthConfig{
		Username: "file_store_user",
		Password: "file_store_pass",
	}
	expectedCredHelperAuth := types.AuthConfig{
		Username: "cred_helper_user",
		Password: "cred_helper_pass",
	}

	configFile := New("filename")
	configFile.CredentialHelpers = map[string]string{testCredHelperRegistryHostname: testCredHelperSuffix}

	testCredHelper := NewMockNativeStore(map[string]types.AuthConfig{
		testCredHelperRegistryHostname: expectedCredHelperAuth,
		// Add in an extra auth entry which doesn't appear in CredentialHelpers section of the configFile.
		// This verifies that only explicitly configured registries are being requested from the cred helpers.
		testExtraCredHelperRegistryHostname: unexpectedCredHelperAuth,
	})

	tmpNewNativeStore := newNativeStore
	defer func() { newNativeStore = tmpNewNativeStore }()
	newNativeStore = func(configFile *ConfigFile, helperSuffix string) credentials.Store {
		return testCredHelper
	}

	authConfigs, err := configFile.GetAllCredentials()
	assert.NilError(t, err)

	expected := make(map[string]types.AuthConfig)
	expected[testCredHelperRegistryHostname] = expectedCredHelperAuth
	assert.Check(t, is.DeepEqual(expected, authConfigs))
	assert.Check(t, is.Equal(0, testCredHelper.(*mockNativeStore).GetAllCallCount))
}

func TestGetAllCredentialsFileStoreAndCredHelper(t *testing.T) {
	testFileStoreRegistryHostname := "example.com"
	testCredHelperSuffix := "test_cred_helper"
	testCredHelperRegistryHostname := "credhelper.com"

	expectedFileStoreAuth := types.AuthConfig{
		Username: "file_store_user",
		Password: "file_store_pass",
	}
	expectedCredHelperAuth := types.AuthConfig{
		Username: "cred_helper_user",
		Password: "cred_helper_pass",
	}

	configFile := New("filename")
	configFile.CredentialHelpers = map[string]string{testCredHelperRegistryHostname: testCredHelperSuffix}
	configFile.AuthConfigs[testFileStoreRegistryHostname] = expectedFileStoreAuth

	testCredHelper := NewMockNativeStore(map[string]types.AuthConfig{testCredHelperRegistryHostname: expectedCredHelperAuth})

	newNativeStore = func(configFile *ConfigFile, helperSuffix string) credentials.Store {
		return testCredHelper
	}

	tmpNewNativeStore := newNativeStore
	defer func() { newNativeStore = tmpNewNativeStore }()
	authConfigs, err := configFile.GetAllCredentials()
	assert.NilError(t, err)

	expected := make(map[string]types.AuthConfig)
	expected[testFileStoreRegistryHostname] = expectedFileStoreAuth
	expected[testCredHelperRegistryHostname] = expectedCredHelperAuth
	assert.Check(t, is.DeepEqual(expected, authConfigs))
	assert.Check(t, is.Equal(0, testCredHelper.(*mockNativeStore).GetAllCallCount))
}

func TestGetAllCredentialsCredStoreAndCredHelper(t *testing.T) {
	testCredStoreSuffix := "test_creds_store"
	testCredStoreRegistryHostname := "credstore.com"
	testCredHelperSuffix := "test_cred_helper"
	testCredHelperRegistryHostname := "credhelper.com"

	configFile := New("filename")
	configFile.CredentialsStore = testCredStoreSuffix
	configFile.CredentialHelpers = map[string]string{testCredHelperRegistryHostname: testCredHelperSuffix}

	expectedCredStoreAuth := types.AuthConfig{
		Username: "cred_store_user",
		Password: "cred_store_pass",
	}
	expectedCredHelperAuth := types.AuthConfig{
		Username: "cred_helper_user",
		Password: "cred_helper_pass",
	}

	testCredHelper := NewMockNativeStore(map[string]types.AuthConfig{testCredHelperRegistryHostname: expectedCredHelperAuth})
	testCredsStore := NewMockNativeStore(map[string]types.AuthConfig{testCredStoreRegistryHostname: expectedCredStoreAuth})

	tmpNewNativeStore := newNativeStore
	defer func() { newNativeStore = tmpNewNativeStore }()
	newNativeStore = func(configFile *ConfigFile, helperSuffix string) credentials.Store {
		if helperSuffix == testCredHelperSuffix {
			return testCredHelper
		}
		return testCredsStore
	}

	authConfigs, err := configFile.GetAllCredentials()
	assert.NilError(t, err)

	expected := make(map[string]types.AuthConfig)
	expected[testCredStoreRegistryHostname] = expectedCredStoreAuth
	expected[testCredHelperRegistryHostname] = expectedCredHelperAuth
	assert.Check(t, is.DeepEqual(expected, authConfigs))
	assert.Check(t, is.Equal(1, testCredsStore.(*mockNativeStore).GetAllCallCount))
	assert.Check(t, is.Equal(0, testCredHelper.(*mockNativeStore).GetAllCallCount))
}

func TestGetAllCredentialsCredHelperOverridesDefaultStore(t *testing.T) {
	testCredStoreSuffix := "test_creds_store"
	testCredHelperSuffix := "test_cred_helper"
	testRegistryHostname := "example.com"

	configFile := New("filename")
	configFile.CredentialsStore = testCredStoreSuffix
	configFile.CredentialHelpers = map[string]string{testRegistryHostname: testCredHelperSuffix}

	unexpectedCredStoreAuth := types.AuthConfig{
		Username: "cred_store_user",
		Password: "cred_store_pass",
	}
	expectedCredHelperAuth := types.AuthConfig{
		Username: "cred_helper_user",
		Password: "cred_helper_pass",
	}

	testCredHelper := NewMockNativeStore(map[string]types.AuthConfig{testRegistryHostname: expectedCredHelperAuth})
	testCredsStore := NewMockNativeStore(map[string]types.AuthConfig{testRegistryHostname: unexpectedCredStoreAuth})

	tmpNewNativeStore := newNativeStore
	defer func() { newNativeStore = tmpNewNativeStore }()
	newNativeStore = func(configFile *ConfigFile, helperSuffix string) credentials.Store {
		if helperSuffix == testCredHelperSuffix {
			return testCredHelper
		}
		return testCredsStore
	}

	authConfigs, err := configFile.GetAllCredentials()
	assert.NilError(t, err)

	expected := make(map[string]types.AuthConfig)
	expected[testRegistryHostname] = expectedCredHelperAuth
	assert.Check(t, is.DeepEqual(expected, authConfigs))
	assert.Check(t, is.Equal(1, testCredsStore.(*mockNativeStore).GetAllCallCount))
	assert.Check(t, is.Equal(0, testCredHelper.(*mockNativeStore).GetAllCallCount))
}

func TestCheckKubernetesConfigurationRaiseAnErrorOnInvalidValue(t *testing.T) {
	testCases := []struct {
		name        string
		config      *KubernetesConfig
		expectError bool
	}{
		{
			"no kubernetes config is valid",
			nil,
			false,
		},
		{
			"enabled is valid",
			&KubernetesConfig{AllNamespaces: "enabled"},
			false,
		},
		{
			"disabled is valid",
			&KubernetesConfig{AllNamespaces: "disabled"},
			false,
		},
		{
			"empty string is valid",
			&KubernetesConfig{AllNamespaces: ""},
			false,
		},
		{
			"other value is invalid",
			&KubernetesConfig{AllNamespaces: "unknown"},
			true,
		},
	}
	for _, test := range testCases {
		err := checkKubernetesConfiguration(test.config)
		if test.expectError {
			assert.Assert(t, err != nil, test.name)
		} else {
			assert.NilError(t, err, test.name)
		}
	}
}

func TestSave(t *testing.T) {
	configFile := New("test-save")
	defer os.Remove("test-save")
	err := configFile.Save()
	assert.NilError(t, err)
	cfg, err := ioutil.ReadFile("test-save")
	assert.NilError(t, err)
	assert.Check(t, is.Equal(string(cfg), "{\n	\"auths\": {}\n}"))
}

func TestPluginConfig(t *testing.T) {
	configFile := New("test-plugin")
	defer os.Remove("test-plugin")

	type PluginConfig1 struct {
		Data1 string `json:"data1"`
		Data2 int    `json:"data2"`
	}
	type PluginConfig2 struct {
		Data3 string `json:"data3"`
	}
	p1 := PluginConfig1{
		Data1: "some string",
		Data2: 42,
	}
	p2 := PluginConfig2{
		Data3: "some other string",
	}

	plugin1, err := json.MarshalIndent(p1, "", "\t")
	assert.NilError(t, err)
	configFile.Plugins["plugin1"] = plugin1

	plugin2, err := json.MarshalIndent(p2, "", "\t")
	assert.NilError(t, err)
	configFile.Plugins["plugin2"] = plugin2

	// Save a config file with some plugin config
	err = configFile.Save()
	assert.NilError(t, err)

	// Read it back and check it has the expected content
	cfg, err := ioutil.ReadFile("test-plugin")
	assert.NilError(t, err)
	golden.Assert(t, string(cfg), "plugin-config.golden")

	// Load it, resave and check again that the content is
	// preserved through a load/save cycle.
	configFile2 := New("test-plugin2")
	defer os.Remove("test-plugin2")
	assert.NilError(t, configFile2.LoadFromReader(bytes.NewReader(cfg)))
	err = configFile2.Save()
	assert.NilError(t, err)
	cfg, err = ioutil.ReadFile("test-plugin2")
	assert.NilError(t, err)
	golden.Assert(t, string(cfg), "plugin-config.golden")

	// Check that the contents was retained properly
	var p1bis PluginConfig1
	assert.Assert(t, is.Contains(configFile2.Plugins, "plugin1"))
	err = json.Unmarshal(configFile2.Plugins["plugin1"], &p1bis)
	assert.NilError(t, err)
	assert.DeepEqual(t, p1, p1bis)

	var p2bis PluginConfig2
	assert.Assert(t, is.Contains(configFile2.Plugins, "plugin2"))
	err = json.Unmarshal(configFile2.Plugins["plugin2"], &p2bis)
	assert.NilError(t, err)
	assert.DeepEqual(t, p2, p2bis)
}
