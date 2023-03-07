package configfile

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/cli/cli/config/credentials"
	"github.com/docker/cli/cli/config/types"
	"github.com/docker/docker/api/types/mount"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ConfigFile ~/.docker/config.json file info
type ConfigFile struct {
	AuthConfigs          map[string]types.AuthConfig  `json:"auths"`
	HTTPHeaders          map[string]string            `json:"HttpHeaders,omitempty"`
	PsFormat             string                       `json:"psFormat,omitempty"`
	ImagesFormat         string                       `json:"imagesFormat,omitempty"`
	NetworksFormat       string                       `json:"networksFormat,omitempty"`
	PluginsFormat        string                       `json:"pluginsFormat,omitempty"`
	VolumesFormat        string                       `json:"volumesFormat,omitempty"`
	StatsFormat          string                       `json:"statsFormat,omitempty"`
	DetachKeys           string                       `json:"detachKeys,omitempty"`
	CredentialsStore     string                       `json:"credsStore,omitempty"`
	CredentialHelpers    map[string]string            `json:"credHelpers,omitempty"`
	Filename             string                       `json:"-"` // Note: for internal use only
	ServiceInspectFormat string                       `json:"serviceInspectFormat,omitempty"`
	ServicesFormat       string                       `json:"servicesFormat,omitempty"`
	TasksFormat          string                       `json:"tasksFormat,omitempty"`
	SecretFormat         string                       `json:"secretFormat,omitempty"`
	ConfigFormat         string                       `json:"configFormat,omitempty"`
	NodesFormat          string                       `json:"nodesFormat,omitempty"`
	PruneFilters         []string                     `json:"pruneFilters,omitempty"`
	Proxies              map[string]ProxyConfig       `json:"proxies,omitempty"`
	Experimental         string                       `json:"experimental,omitempty"`
	StackOrchestrator    string                       `json:"stackOrchestrator,omitempty"` // Deprecated: swarm is now the default orchestrator, and this option is ignored.
	CurrentContext       string                       `json:"currentContext,omitempty"`
	CLIPluginsExtraDirs  []string                     `json:"cliPluginsExtraDirs,omitempty"`
	Plugins              map[string]map[string]string `json:"plugins,omitempty"`
	Aliases              map[string]string            `json:"aliases,omitempty"`
	BindMaps             map[string]map[string]string `json:"bindMaps,omitempty"`
}

// ProxyConfig contains proxy configuration settings
type ProxyConfig struct {
	HTTPProxy  string `json:"httpProxy,omitempty"`
	HTTPSProxy string `json:"httpsProxy,omitempty"`
	NoProxy    string `json:"noProxy,omitempty"`
	FTPProxy   string `json:"ftpProxy,omitempty"`
	AllProxy   string `json:"allProxy,omitempty"`
}

// New initializes an empty configuration file for the given filename 'fn'
func New(fn string) *ConfigFile {
	return &ConfigFile{
		AuthConfigs: make(map[string]types.AuthConfig),
		HTTPHeaders: make(map[string]string),
		Filename:    fn,
		Plugins:     make(map[string]map[string]string),
		Aliases:     make(map[string]string),
	}
}

// LoadFromReader reads the configuration data given and sets up the auth config
// information with given directory and populates the receiver object
func (configFile *ConfigFile) LoadFromReader(configData io.Reader) error {
	if err := json.NewDecoder(configData).Decode(configFile); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	var err error
	for addr, ac := range configFile.AuthConfigs {
		if ac.Auth != "" {
			ac.Username, ac.Password, err = decodeAuth(ac.Auth)
			if err != nil {
				return err
			}
		}
		ac.Auth = ""
		ac.ServerAddress = addr
		configFile.AuthConfigs[addr] = ac
	}
	return nil
}

// ContainsAuth returns whether there is authentication configured
// in this file or not.
func (configFile *ConfigFile) ContainsAuth() bool {
	return configFile.CredentialsStore != "" ||
		len(configFile.CredentialHelpers) > 0 ||
		len(configFile.AuthConfigs) > 0
}

// GetAuthConfigs returns the mapping of repo to auth configuration
func (configFile *ConfigFile) GetAuthConfigs() map[string]types.AuthConfig {
	return configFile.AuthConfigs
}

// SaveToWriter encodes and writes out all the authorization information to
// the given writer
func (configFile *ConfigFile) SaveToWriter(writer io.Writer) error {
	// Encode sensitive data into a new/temp struct
	tmpAuthConfigs := make(map[string]types.AuthConfig, len(configFile.AuthConfigs))
	for k, authConfig := range configFile.AuthConfigs {
		authCopy := authConfig
		// encode and save the authstring, while blanking out the original fields
		authCopy.Auth = encodeAuth(&authCopy)
		authCopy.Username = ""
		authCopy.Password = ""
		authCopy.ServerAddress = ""
		tmpAuthConfigs[k] = authCopy
	}

	saveAuthConfigs := configFile.AuthConfigs
	configFile.AuthConfigs = tmpAuthConfigs
	defer func() { configFile.AuthConfigs = saveAuthConfigs }()

	// User-Agent header is automatically set, and should not be stored in the configuration
	for v := range configFile.HTTPHeaders {
		if strings.EqualFold(v, "User-Agent") {
			delete(configFile.HTTPHeaders, v)
		}
	}

	data, err := json.MarshalIndent(configFile, "", "\t")
	if err != nil {
		return err
	}
	_, err = writer.Write(data)
	return err
}

// Save encodes and writes out all the authorization information
func (configFile *ConfigFile) Save() (retErr error) {
	if configFile.Filename == "" {
		return errors.Errorf("Can't save config with empty filename")
	}

	dir := filepath.Dir(configFile.Filename)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	temp, err := os.CreateTemp(dir, filepath.Base(configFile.Filename))
	if err != nil {
		return err
	}
	defer func() {
		temp.Close()
		if retErr != nil {
			if err := os.Remove(temp.Name()); err != nil {
				logrus.WithError(err).WithField("file", temp.Name()).Debug("Error cleaning up temp file")
			}
		}
	}()

	err = configFile.SaveToWriter(temp)
	if err != nil {
		return err
	}

	if err := temp.Close(); err != nil {
		return errors.Wrap(err, "error closing temp file")
	}

	// Handle situation where the configfile is a symlink
	cfgFile := configFile.Filename
	if f, err := os.Readlink(cfgFile); err == nil {
		cfgFile = f
	}

	// Try copying the current config file (if any) ownership and permissions
	copyFilePermissions(cfgFile, temp.Name())
	return os.Rename(temp.Name(), cfgFile)
}

// ParseProxyConfig computes proxy configuration by retrieving the config for the provided host and
// then checking this against any environment variables provided to the container
func (configFile *ConfigFile) ParseProxyConfig(host string, runOpts map[string]*string) map[string]*string {
	var cfgKey string

	if _, ok := configFile.Proxies[host]; !ok {
		cfgKey = "default"
	} else {
		cfgKey = host
	}

	config := configFile.Proxies[cfgKey]
	permitted := map[string]*string{
		"HTTP_PROXY":  &config.HTTPProxy,
		"HTTPS_PROXY": &config.HTTPSProxy,
		"NO_PROXY":    &config.NoProxy,
		"FTP_PROXY":   &config.FTPProxy,
		"ALL_PROXY":   &config.AllProxy,
	}
	m := runOpts
	if m == nil {
		m = make(map[string]*string)
	}
	for k := range permitted {
		if *permitted[k] == "" {
			continue
		}
		if _, ok := m[k]; !ok {
			m[k] = permitted[k]
		}
		if _, ok := m[strings.ToLower(k)]; !ok {
			m[strings.ToLower(k)] = permitted[k]
		}
	}
	return m
}

// ApplyBindMap updates bind mount options in-place with `bindMap` configurations
func (configFile *ConfigFile) ApplyBindMap(host string, binds []string) []string {
	maps, ok := configFile.BindMaps[host]
	if !ok {
		maps = configFile.BindMaps["default"]
	}
	if len(maps) == 0 {
		return binds
	}
	for index, bind := range binds {
		for sourcePrefix, destPrefix := range maps {
			newBind := mapBind(bind, sourcePrefix, destPrefix)
			if newBind != "" {
				binds[index] = newBind
				break
			}
		}
	}
	return binds
}

// ApplyBindMapToMounts updates mount options with bind type in-place with `bindMap` configurations
func (configFile *ConfigFile) ApplyBindMapToMounts(host string, mounts []mount.Mount) []mount.Mount {
	maps, ok := configFile.BindMaps[host]
	if !ok {
		maps = configFile.BindMaps["default"]
	}
	if len(maps) == 0 {
		return mounts
	}
	for index, mountSpec := range mounts {
		if mountSpec.Type != mount.TypeBind {
			continue
		}
		for sourcePrefix, destPrefix := range maps {
			newBind := mapBindSource(mountSpec.Source, sourcePrefix, destPrefix)
			if newBind != "" {
				mounts[index].Source = newBind
				break
			}
		}
	}
	return mounts
}

func isWindowsAbsolutePath(path string) bool {
	if len(path) < 3 {
		return false
	}
	return path[1] == ':' &&
		path[2] == '\\' &&
		(('a' <= path[0] && path[0] <= 'z') || ('A' <= path[0] && path[0] <= 'Z'))
}

func mapBind(bind, sourcePrefix, destPrefix string) string {
	if len(sourcePrefix) == 0 || len(destPrefix) == 0 {
		return ""
	}

	isWindows := isWindowsAbsolutePath(bind)
	var source string
	var rest string
	if isWindows {
		// bind specification must be like "C:\windows:/bind:ro"
		arr := strings.SplitN(bind, ":", 3)
		if len(arr) < 3 {
			// invalid specification
			return ""
		}
		source = arr[0] + ":" + arr[1]
		rest = arr[2]
	} else {
		// bind specification must be like "/home/user/path:/bind:ro"
		arr := strings.SplitN(bind, ":", 2)
		if len(arr) < 2 {
			// invalid specification
			return ""
		}
		source = arr[0]
		rest = arr[1]
	}

	source = mapBindSourceWithPlatform(source, sourcePrefix, destPrefix, isWindows)
	if source == "" {
		return ""
	}

	return source + ":" + rest
}

func mapBindSource(source, sourcePrefix, destPrefix string) string {
	if len(sourcePrefix) == 0 || len(destPrefix) == 0 {
		return ""
	}
	return mapBindSourceWithPlatform(
		source,
		sourcePrefix,
		destPrefix,
		isWindowsAbsolutePath(source),
	)
}

func mapBindSourceWithPlatform(source, sourcePrefix, destPrefix string, isWindows bool) string {
	// precondition: sourcePrefix and destPrefix must not be empty.
	// normalize prefixes
	if (!isWindows && sourcePrefix[len(sourcePrefix)-1] == '/') ||
		(isWindows && sourcePrefix[len(sourcePrefix)-1] == '\\') {
		sourcePrefix = sourcePrefix[0 : len(sourcePrefix)-1]
	}

	if !strings.HasPrefix(source, sourcePrefix) {
		return ""
	}

	if destPrefix[len(destPrefix)-1] == '/' {
		destPrefix = destPrefix[0 : len(destPrefix)-1]
	}

	// Replace sourcePrefix with destPrefix.
	// Note: destPrefix is always assumed Unix-like paths here.
	rest := source[len(sourcePrefix):]

	// Ignore if matched with non-separator boundary
	// Example:
	// * source: /pathspecification/...
	// * sourcePrefix: /path
	if len(rest) != 0 &&
		!((!isWindows && rest[0] == '/') || (isWindows && rest[0] == '\\')) {
		return ""
	}

	if !isWindows {
		// can be simply replaced.
		return destPrefix + rest
	}
	return destPrefix + strings.ReplaceAll(rest, "\\", "/")
}

// encodeAuth creates a base64 encoded string to containing authorization information
func encodeAuth(authConfig *types.AuthConfig) string {
	if authConfig.Username == "" && authConfig.Password == "" {
		return ""
	}

	authStr := authConfig.Username + ":" + authConfig.Password
	msg := []byte(authStr)
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(msg)))
	base64.StdEncoding.Encode(encoded, msg)
	return string(encoded)
}

// decodeAuth decodes a base64 encoded string and returns username and password
func decodeAuth(authStr string) (string, string, error) {
	if authStr == "" {
		return "", "", nil
	}

	decLen := base64.StdEncoding.DecodedLen(len(authStr))
	decoded := make([]byte, decLen)
	authByte := []byte(authStr)
	n, err := base64.StdEncoding.Decode(decoded, authByte)
	if err != nil {
		return "", "", err
	}
	if n > decLen {
		return "", "", errors.Errorf("Something went wrong decoding auth config")
	}
	userName, password, ok := strings.Cut(string(decoded), ":")
	if !ok || userName == "" {
		return "", "", errors.Errorf("Invalid auth configuration file")
	}
	return userName, strings.Trim(password, "\x00"), nil
}

// GetCredentialsStore returns a new credentials store from the settings in the
// configuration file
func (configFile *ConfigFile) GetCredentialsStore(registryHostname string) credentials.Store {
	if helper := getConfiguredCredentialStore(configFile, registryHostname); helper != "" {
		return newNativeStore(configFile, helper)
	}
	return credentials.NewFileStore(configFile)
}

// var for unit testing.
var newNativeStore = func(configFile *ConfigFile, helperSuffix string) credentials.Store {
	return credentials.NewNativeStore(configFile, helperSuffix)
}

// GetAuthConfig for a repository from the credential store
func (configFile *ConfigFile) GetAuthConfig(registryHostname string) (types.AuthConfig, error) {
	return configFile.GetCredentialsStore(registryHostname).Get(registryHostname)
}

// getConfiguredCredentialStore returns the credential helper configured for the
// given registry, the default credsStore, or the empty string if neither are
// configured.
func getConfiguredCredentialStore(c *ConfigFile, registryHostname string) string {
	if c.CredentialHelpers != nil && registryHostname != "" {
		if helper, exists := c.CredentialHelpers[registryHostname]; exists {
			return helper
		}
	}
	return c.CredentialsStore
}

// GetAllCredentials returns all of the credentials stored in all of the
// configured credential stores.
func (configFile *ConfigFile) GetAllCredentials() (map[string]types.AuthConfig, error) {
	auths := make(map[string]types.AuthConfig)
	addAll := func(from map[string]types.AuthConfig) {
		for reg, ac := range from {
			auths[reg] = ac
		}
	}

	defaultStore := configFile.GetCredentialsStore("")
	newAuths, err := defaultStore.GetAll()
	if err != nil {
		return nil, err
	}
	addAll(newAuths)

	// Auth configs from a registry-specific helper should override those from the default store.
	for registryHostname := range configFile.CredentialHelpers {
		newAuth, err := configFile.GetAuthConfig(registryHostname)
		if err != nil {
			logrus.WithError(err).Warnf("Failed to get credentials for registry: %s", registryHostname)
			continue
		}
		auths[registryHostname] = newAuth
	}
	return auths, nil
}

// GetFilename returns the file name that this config file is based on.
func (configFile *ConfigFile) GetFilename() string {
	return configFile.Filename
}

// PluginConfig retrieves the requested option for the given plugin.
func (configFile *ConfigFile) PluginConfig(pluginname, option string) (string, bool) {
	if configFile.Plugins == nil {
		return "", false
	}
	pluginConfig, ok := configFile.Plugins[pluginname]
	if !ok {
		return "", false
	}
	value, ok := pluginConfig[option]
	return value, ok
}

// SetPluginConfig sets the option to the given value for the given
// plugin. Passing a value of "" will remove the option. If removing
// the final config item for a given plugin then also cleans up the
// overall plugin entry.
func (configFile *ConfigFile) SetPluginConfig(pluginname, option, value string) {
	if configFile.Plugins == nil {
		configFile.Plugins = make(map[string]map[string]string)
	}
	pluginConfig, ok := configFile.Plugins[pluginname]
	if !ok {
		pluginConfig = make(map[string]string)
		configFile.Plugins[pluginname] = pluginConfig
	}
	if value != "" {
		pluginConfig[option] = value
	} else {
		delete(pluginConfig, option)
	}
	if len(pluginConfig) == 0 {
		delete(configFile.Plugins, pluginname)
	}
}
