package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/credentials"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/skip"
)

func setupConfigDir(t *testing.T) string {
	t.Helper()
	tmpdir := t.TempDir()
	oldDir := Dir()
	SetDir(tmpdir)
	t.Cleanup(func() {
		SetDir(oldDir)
	})
	return tmpdir
}

func TestEmptyConfigDir(t *testing.T) {
	tmpHome := setupConfigDir(t)

	config, err := Load("")
	assert.NilError(t, err)

	expectedConfigFilename := filepath.Join(tmpHome, ConfigFileName)
	assert.Check(t, is.Equal(expectedConfigFilename, config.Filename))

	// Now save it and make sure it shows up in new form
	saveConfigAndValidateNewFormat(t, config, tmpHome)
}

func TestMissingFile(t *testing.T) {
	tmpHome := t.TempDir()

	config, err := Load(tmpHome)
	assert.NilError(t, err)

	// Now save it and make sure it shows up in new form
	saveConfigAndValidateNewFormat(t, config, tmpHome)
}

// TestLoadDanglingSymlink verifies that we gracefully handle dangling symlinks.
//
// TODO(thaJeztah): consider whether we want dangling symlinks to be an error condition instead.
func TestLoadDanglingSymlink(t *testing.T) {
	cfgDir := t.TempDir()
	cfgFile := filepath.Join(cfgDir, ConfigFileName)
	err := os.Symlink(filepath.Join(cfgDir, "no-such-file"), cfgFile)
	assert.NilError(t, err)

	config, err := Load(cfgDir)
	assert.NilError(t, err)

	// Now save it and make sure it shows up in new form
	saveConfigAndValidateNewFormat(t, config, cfgDir)

	// Make sure we kept the symlink.
	fi, err := os.Lstat(cfgFile)
	assert.NilError(t, err)
	assert.Equal(t, fi.Mode()&os.ModeSymlink, os.ModeSymlink, "expected %v to be a symlink", cfgFile)
}

func TestLoadNoPermissions(t *testing.T) {
	if runtime.GOOS != "windows" {
		skip.If(t, os.Getuid() == 0, "cannot test permission denied when running as root")
	}
	cfgDir := t.TempDir()
	cfgFile := filepath.Join(cfgDir, ConfigFileName)
	err := os.WriteFile(cfgFile, []byte(`{}`), os.FileMode(0o000))
	assert.NilError(t, err)

	_, err = Load(cfgDir)
	assert.ErrorIs(t, err, os.ErrPermission)
}

func TestSaveFileToDirs(t *testing.T) {
	tmpHome := filepath.Join(t.TempDir(), ".docker")
	config, err := Load(tmpHome)
	assert.NilError(t, err)

	// Now save it and make sure it shows up in new form
	saveConfigAndValidateNewFormat(t, config, tmpHome)
}

func TestEmptyFile(t *testing.T) {
	tmpHome := t.TempDir()

	fn := filepath.Join(tmpHome, ConfigFileName)
	err := os.WriteFile(fn, []byte(""), 0o600)
	assert.NilError(t, err)

	_, err = Load(tmpHome)
	assert.NilError(t, err)
}

func TestEmptyJSON(t *testing.T) {
	tmpHome := t.TempDir()

	fn := filepath.Join(tmpHome, ConfigFileName)
	err := os.WriteFile(fn, []byte("{}"), 0o600)
	assert.NilError(t, err)

	config, err := Load(tmpHome)
	assert.NilError(t, err)

	// Now save it and make sure it shows up in new form
	saveConfigAndValidateNewFormat(t, config, tmpHome)
}

func TestMalformedJSON(t *testing.T) {
	tmpHome := t.TempDir()

	fn := filepath.Join(tmpHome, ConfigFileName)
	err := os.WriteFile(fn, []byte("{"), 0o600)
	assert.NilError(t, err)

	_, err = Load(tmpHome)
	assert.Check(t, is.ErrorContains(err, fmt.Sprintf(`parsing config file (%s):`, fn)))
}

func TestNewJSON(t *testing.T) {
	tmpHome := t.TempDir()

	fn := filepath.Join(tmpHome, ConfigFileName)
	js := ` { "auths": { "https://index.docker.io/v1/": { "auth": "am9lam9lOmhlbGxv" } } }`
	if err := os.WriteFile(fn, []byte(js), 0o600); err != nil {
		t.Fatal(err)
	}

	config, err := Load(tmpHome)
	assert.NilError(t, err)

	ac := config.AuthConfigs["https://index.docker.io/v1/"]
	assert.Equal(t, ac.Username, "joejoe")
	assert.Equal(t, ac.Password, "hello")

	// Now save it and make sure it shows up in new form
	configStr := saveConfigAndValidateNewFormat(t, config, tmpHome)

	expConfStr := `{
	"auths": {
		"https://index.docker.io/v1/": {
			"auth": "am9lam9lOmhlbGxv"
		}
	}
}`

	if configStr != expConfStr {
		t.Fatalf("Should have save in new form: \n%s\n not \n%s", configStr, expConfStr)
	}
}

func TestNewJSONNoEmail(t *testing.T) {
	tmpHome := t.TempDir()

	fn := filepath.Join(tmpHome, ConfigFileName)
	js := ` { "auths": { "https://index.docker.io/v1/": { "auth": "am9lam9lOmhlbGxv" } } }`
	if err := os.WriteFile(fn, []byte(js), 0o600); err != nil {
		t.Fatal(err)
	}

	config, err := Load(tmpHome)
	assert.NilError(t, err)

	ac := config.AuthConfigs["https://index.docker.io/v1/"]
	assert.Equal(t, ac.Username, "joejoe")
	assert.Equal(t, ac.Password, "hello")

	// Now save it and make sure it shows up in new form
	configStr := saveConfigAndValidateNewFormat(t, config, tmpHome)

	expConfStr := `{
	"auths": {
		"https://index.docker.io/v1/": {
			"auth": "am9lam9lOmhlbGxv"
		}
	}
}`

	if configStr != expConfStr {
		t.Fatalf("Should have save in new form: \n%s\n not \n%s", configStr, expConfStr)
	}
}

func TestJSONWithPsFormat(t *testing.T) {
	tmpHome := t.TempDir()

	fn := filepath.Join(tmpHome, ConfigFileName)
	js := `{
		"auths": { "https://index.docker.io/v1/": { "auth": "am9lam9lOmhlbGxv", "email": "user@example.com" } },
		"psFormat": "table {{.ID}}\\t{{.Label \"com.docker.label.cpu\"}}"
}`
	if err := os.WriteFile(fn, []byte(js), 0o600); err != nil {
		t.Fatal(err)
	}

	config, err := Load(tmpHome)
	assert.NilError(t, err)

	if config.PsFormat != `table {{.ID}}\t{{.Label "com.docker.label.cpu"}}` {
		t.Fatalf("Unknown ps format: %s\n", config.PsFormat)
	}

	// Now save it and make sure it shows up in new form
	configStr := saveConfigAndValidateNewFormat(t, config, tmpHome)
	if !strings.Contains(configStr, `"psFormat":`) ||
		!strings.Contains(configStr, "{{.ID}}") {
		t.Fatalf("Should have save in new form: %s", configStr)
	}
}

func TestJSONWithCredentialStore(t *testing.T) {
	tmpHome := t.TempDir()

	fn := filepath.Join(tmpHome, ConfigFileName)
	js := `{
		"auths": { "https://index.docker.io/v1/": { "auth": "am9lam9lOmhlbGxv", "email": "user@example.com" } },
		"credsStore": "crazy-secure-storage"
}`
	if err := os.WriteFile(fn, []byte(js), 0o600); err != nil {
		t.Fatal(err)
	}

	config, err := Load(tmpHome)
	assert.NilError(t, err)

	if config.CredentialsStore != "crazy-secure-storage" {
		t.Fatalf("Unknown credential store: %s\n", config.CredentialsStore)
	}

	// Now save it and make sure it shows up in new form
	configStr := saveConfigAndValidateNewFormat(t, config, tmpHome)
	if !strings.Contains(configStr, `"credsStore":`) ||
		!strings.Contains(configStr, "crazy-secure-storage") {
		t.Fatalf("Should have save in new form: %s", configStr)
	}
}

func TestJSONWithCredentialHelpers(t *testing.T) {
	tmpHome := t.TempDir()

	fn := filepath.Join(tmpHome, ConfigFileName)
	js := `{
		"auths": { "https://index.docker.io/v1/": { "auth": "am9lam9lOmhlbGxv", "email": "user@example.com" } },
		"credHelpers": { "images.io": "images-io", "containers.com": "crazy-secure-storage" }
}`
	if err := os.WriteFile(fn, []byte(js), 0o600); err != nil {
		t.Fatal(err)
	}

	config, err := Load(tmpHome)
	assert.NilError(t, err)

	if config.CredentialHelpers == nil {
		t.Fatal("config.CredentialHelpers was nil")
	} else if config.CredentialHelpers["images.io"] != "images-io" ||
		config.CredentialHelpers["containers.com"] != "crazy-secure-storage" {
		t.Fatalf("Credential helpers not deserialized properly: %v\n", config.CredentialHelpers)
	}

	// Now save it and make sure it shows up in new form
	configStr := saveConfigAndValidateNewFormat(t, config, tmpHome)
	if !strings.Contains(configStr, `"credHelpers":`) ||
		!strings.Contains(configStr, "images.io") ||
		!strings.Contains(configStr, "images-io") ||
		!strings.Contains(configStr, "containers.com") ||
		!strings.Contains(configStr, "crazy-secure-storage") {
		t.Fatalf("Should have save in new form: %s", configStr)
	}
}

// Save it and make sure it shows up in new form
func saveConfigAndValidateNewFormat(t *testing.T, config *configfile.ConfigFile, configDir string) string {
	t.Helper()
	assert.NilError(t, config.Save())

	buf, err := os.ReadFile(filepath.Join(configDir, ConfigFileName))
	assert.NilError(t, err)
	assert.Check(t, is.Contains(string(buf), `"auths":`))
	return string(buf)
}

func TestConfigDir(t *testing.T) {
	tmpHome := t.TempDir()

	if Dir() == tmpHome {
		t.Fatalf("Expected ConfigDir to be different than %s by default, but was the same", tmpHome)
	}

	// Update configDir
	SetDir(tmpHome)

	if Dir() != tmpHome {
		t.Fatalf("Expected ConfigDir to %s, but was %s", tmpHome, Dir())
	}
}

func TestJSONReaderNoFile(t *testing.T) {
	js := ` { "auths": { "https://index.docker.io/v1/": { "auth": "am9lam9lOmhlbGxv", "email": "user@example.com" } } }`

	config, err := LoadFromReader(strings.NewReader(js))
	assert.NilError(t, err)

	ac := config.AuthConfigs["https://index.docker.io/v1/"]
	assert.Equal(t, ac.Username, "joejoe")
	assert.Equal(t, ac.Password, "hello")
}

func TestJSONWithPsFormatNoFile(t *testing.T) {
	js := `{
		"auths": { "https://index.docker.io/v1/": { "auth": "am9lam9lOmhlbGxv", "email": "user@example.com" } },
		"psFormat": "table {{.ID}}\\t{{.Label \"com.docker.label.cpu\"}}"
}`
	config, err := LoadFromReader(strings.NewReader(js))
	assert.NilError(t, err)

	if config.PsFormat != `table {{.ID}}\t{{.Label "com.docker.label.cpu"}}` {
		t.Fatalf("Unknown ps format: %s\n", config.PsFormat)
	}
}

func TestJSONSaveWithNoFile(t *testing.T) {
	js := `{
		"auths": { "https://index.docker.io/v1/": { "auth": "am9lam9lOmhlbGxv" } },
		"psFormat": "table {{.ID}}\\t{{.Label \"com.docker.label.cpu\"}}"
}`
	config, err := LoadFromReader(strings.NewReader(js))
	assert.NilError(t, err)
	err = config.Save()
	assert.ErrorContains(t, err, "with empty filename")

	tmpHome := t.TempDir()

	fn := filepath.Join(tmpHome, ConfigFileName)
	f, _ := os.OpenFile(fn, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	defer f.Close()

	assert.NilError(t, config.SaveToWriter(f))
	buf, err := os.ReadFile(filepath.Join(tmpHome, ConfigFileName))
	assert.NilError(t, err)
	expConfStr := `{
	"auths": {
		"https://index.docker.io/v1/": {
			"auth": "am9lam9lOmhlbGxv"
		}
	},
	"psFormat": "table {{.ID}}\\t{{.Label \"com.docker.label.cpu\"}}"
}`
	if string(buf) != expConfStr {
		t.Fatalf("Should have save in new form: \n%s\nnot \n%s", string(buf), expConfStr)
	}
}

func TestLoadDefaultConfigFile(t *testing.T) {
	dir := setupConfigDir(t)
	buffer := new(bytes.Buffer)

	filename := filepath.Join(dir, ConfigFileName)
	content := []byte(`{"PsFormat": "format"}`)
	err := os.WriteFile(filename, content, 0o644)
	assert.NilError(t, err)

	t.Run("success", func(t *testing.T) {
		configFile := LoadDefaultConfigFile(buffer)
		credStore := credentials.DetectDefaultStore("")
		expected := configfile.New(filename)
		expected.CredentialsStore = credStore
		expected.PsFormat = "format"

		assert.Check(t, is.DeepEqual(expected, configFile))
		assert.Check(t, is.Equal(buffer.String(), ""))
	})

	t.Run("permission error", func(t *testing.T) {
		if runtime.GOOS != "windows" {
			skip.If(t, os.Getuid() == 0, "cannot test permission denied when running as root")
		}
		err = os.Chmod(filename, 0o000)
		assert.NilError(t, err)

		buffer.Reset()
		_ = LoadDefaultConfigFile(buffer)
		warnings := buffer.String()

		assert.Check(t, is.Contains(warnings, "WARNING:"))
		assert.Check(t, is.Contains(warnings, os.ErrPermission.Error()))
	})
}

func TestConfigPath(t *testing.T) {
	oldDir := Dir()

	for _, tc := range []struct {
		name        string
		dir         string
		path        []string
		expected    string
		expectedErr string
	}{
		{
			name:     "valid_path",
			dir:      "dummy",
			path:     []string{"a", "b"},
			expected: filepath.Join("dummy", "a", "b"),
		},
		{
			name:     "valid_path_absolute_dir",
			dir:      "/dummy",
			path:     []string{"a", "b"},
			expected: filepath.Join("/dummy", "a", "b"),
		},
		{
			name:        "invalid_relative_path",
			dir:         "dummy",
			path:        []string{"e", "..", "..", "f"},
			expectedErr: fmt.Sprintf("is outside of root config directory %q", "dummy"),
		},
		{
			name:        "invalid_absolute_path",
			dir:         "dummy",
			path:        []string{"/a", "..", ".."},
			expectedErr: fmt.Sprintf("is outside of root config directory %q", "dummy"),
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			SetDir(tc.dir)
			f, err := Path(tc.path...)
			assert.Equal(t, f, tc.expected)
			if tc.expectedErr == "" {
				assert.NilError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.expectedErr)
			}
		})
	}

	SetDir(oldDir)
}

// TestSetDir verifies that Dir() does not overwrite the value set through
// SetDir() if it has not been run before.
func TestSetDir(t *testing.T) {
	const expected = "my_config_dir"
	resetConfigDir()
	SetDir(expected)
	assert.Check(t, is.Equal(Dir(), expected))
}
