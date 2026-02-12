package manager

import (
	"context"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

type fakeConfigProvider struct {
	cfg *configfile.ConfigFile
}

func (f *fakeConfigProvider) ConfigFile() *configfile.ConfigFile {
	return f.cfg
}

func TestGetNaiveFlags(t *testing.T) {
	testCases := []struct {
		args          []string
		expectedFlags map[string]string
	}{
		{
			args:          []string{"docker"},
			expectedFlags: map[string]string{},
		},
		{
			args: []string{"docker", "build", "-q", "--file", "test.Dockerfile", "."},
			expectedFlags: map[string]string{
				"q":    "",
				"file": "",
			},
		},
		{
			args: []string{"docker", "--context", "a-context", "pull", "-q", "--progress", "auto", "alpine"},
			expectedFlags: map[string]string{
				"context":  "",
				"q":        "",
				"progress": "",
			},
		},
	}

	for _, tc := range testCases {
		assert.DeepEqual(t, getNaiveFlags(tc.args), tc.expectedFlags)
	}
}

func TestPluginMatch(t *testing.T) {
	testCases := []struct {
		doc             string
		commandString   string
		pluginConfig    map[string]string
		cmdErrorMessage string
		expectedMatch   string
		expectedOk      bool
	}{
		{
			doc:           "hooks prefix match",
			commandString: "image ls",
			pluginConfig: map[string]string{
				"hooks": "image",
			},
			expectedMatch: "image",
			expectedOk:    true,
		},
		{
			doc:           "hooks no match",
			commandString: "context ls",
			pluginConfig: map[string]string{
				"hooks": "build",
			},
			expectedMatch: "",
			expectedOk:    false,
		},
		{
			doc:           "hooks exact match",
			commandString: "context ls",
			pluginConfig: map[string]string{
				"hooks": "context ls",
			},
			expectedMatch: "context ls",
			expectedOk:    true,
		},
		{
			doc:           "hooks first match wins",
			commandString: "image ls",
			pluginConfig: map[string]string{
				"hooks": "image ls,image",
			},
			expectedMatch: "image ls",
			expectedOk:    true,
		},
		{
			doc:           "hooks empty string",
			commandString: "image ls",
			pluginConfig: map[string]string{
				"hooks": "",
			},
			expectedMatch: "",
			expectedOk:    false,
		},
		{
			doc:           "hooks partial token no match",
			commandString: "image inspect",
			pluginConfig: map[string]string{
				"hooks": "image i",
			},
			expectedMatch: "",
			expectedOk:    false,
		},
		{
			doc:           "hooks prefix token match",
			commandString: "image inspect",
			pluginConfig: map[string]string{
				"hooks": "image",
			},
			expectedMatch: "image",
			expectedOk:    true,
		},
		{
			doc:           "error-hooks match on error",
			commandString: "build",
			pluginConfig: map[string]string{
				"error-hooks": "build",
			},
			cmdErrorMessage: "exit status 1",
			expectedMatch:   "build",
			expectedOk:      true,
		},
		{
			doc:           "error-hooks no match on success",
			commandString: "build",
			pluginConfig: map[string]string{
				"error-hooks": "build",
			},
			cmdErrorMessage: "",
			expectedMatch:   "",
			expectedOk:      false,
		},
		{
			doc:           "error-hooks prefix match on error",
			commandString: "compose up",
			pluginConfig: map[string]string{
				"error-hooks": "compose",
			},
			cmdErrorMessage: "exit status 1",
			expectedMatch:   "compose",
			expectedOk:      true,
		},
		{
			doc:           "error-hooks no match for wrong command",
			commandString: "pull",
			pluginConfig: map[string]string{
				"error-hooks": "build",
			},
			cmdErrorMessage: "exit status 1",
			expectedMatch:   "",
			expectedOk:      false,
		},
		{
			doc:           "hooks takes precedence over error-hooks",
			commandString: "build",
			pluginConfig: map[string]string{
				"hooks":       "build",
				"error-hooks": "build",
			},
			cmdErrorMessage: "exit status 1",
			expectedMatch:   "build",
			expectedOk:      true,
		},
		{
			doc:           "hooks fires on success even with error-hooks configured",
			commandString: "build",
			pluginConfig: map[string]string{
				"hooks":       "build",
				"error-hooks": "build",
			},
			cmdErrorMessage: "",
			expectedMatch:   "build",
			expectedOk:      true,
		},
		{
			doc:           "error-hooks with multiple commands",
			commandString: "compose up",
			pluginConfig: map[string]string{
				"error-hooks": "build,compose up,pull",
			},
			cmdErrorMessage: "exit status 1",
			expectedMatch:   "compose up",
			expectedOk:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.doc, func(t *testing.T) {
			match, ok := pluginMatch(tc.pluginConfig, tc.commandString, tc.cmdErrorMessage)
			assert.Equal(t, ok, tc.expectedOk)
			assert.Equal(t, match, tc.expectedMatch)
		})
	}
}

func TestMatchHookConfig(t *testing.T) {
	testCases := []struct {
		doc             string
		configuredHooks string
		subCmd          string
		expectedMatch   string
		expectedOk      bool
	}{
		{
			doc:             "empty config",
			configuredHooks: "",
			subCmd:          "build",
			expectedMatch:   "",
			expectedOk:      false,
		},
		{
			doc:             "exact match",
			configuredHooks: "build",
			subCmd:          "build",
			expectedMatch:   "build",
			expectedOk:      true,
		},
		{
			doc:             "prefix match",
			configuredHooks: "image",
			subCmd:          "image ls",
			expectedMatch:   "image",
			expectedOk:      true,
		},
		{
			doc:             "comma-separated match",
			configuredHooks: "pull,build,push",
			subCmd:          "build",
			expectedMatch:   "build",
			expectedOk:      true,
		},
		{
			doc:             "no match",
			configuredHooks: "pull,push",
			subCmd:          "build",
			expectedMatch:   "",
			expectedOk:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.doc, func(t *testing.T) {
			match, ok := matchHookConfig(tc.configuredHooks, tc.subCmd)
			assert.Equal(t, ok, tc.expectedOk)
			assert.Equal(t, match, tc.expectedMatch)
		})
	}
}

func TestAppendNextSteps(t *testing.T) {
	testCases := []struct {
		processed   []string
		expectedOut []string
	}{
		{
			processed:   []string{},
			expectedOut: []string{},
		},
		{
			processed:   []string{"", ""},
			expectedOut: []string{},
		},
		{
			processed:   []string{"Some hint", "", "Some other hint"},
			expectedOut: []string{"Some hint", "", "Some other hint"},
		},
		{
			processed:   []string{"Hint 1", "Hint 2"},
			expectedOut: []string{"Hint 1", "Hint 2"},
		},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			got, appended := appendNextSteps([]string{}, tc.processed)
			assert.Check(t, is.DeepEqual(got, tc.expectedOut))
			assert.Check(t, is.Equal(appended, len(got) > 0))
		})
	}
}

func TestRunPluginHooksPassesErrorMessage(t *testing.T) {
	cfg := configfile.New("")
	cfg.Plugins = map[string]map[string]string{
		"test-plugin": {"hooks": "build"},
	}
	provider := &fakeConfigProvider{cfg: cfg}
	root := &cobra.Command{Use: "docker"}
	sub := &cobra.Command{Use: "build"}
	root.AddCommand(sub)

	// Should not panic with empty error message (success case)
	RunPluginHooks(context.Background(), provider, root, sub, []string{"build"}, "")

	// Should not panic with non-empty error message (failure case)
	RunPluginHooks(context.Background(), provider, root, sub, []string{"build"}, "exit status 1")
}

func TestRunPluginHooksErrorHooks(t *testing.T) {
	cfg := configfile.New("")
	cfg.Plugins = map[string]map[string]string{
		"test-plugin": {"error-hooks": "build"},
	}
	provider := &fakeConfigProvider{cfg: cfg}
	root := &cobra.Command{Use: "docker"}
	sub := &cobra.Command{Use: "build"}
	root.AddCommand(sub)

	// Should not panic — error-hooks with error message
	RunPluginHooks(context.Background(), provider, root, sub, []string{"build"}, "exit status 1")

	// Should not panic — error-hooks with no error (should be skipped)
	RunPluginHooks(context.Background(), provider, root, sub, []string{"build"}, "")
}

func TestInvokeAndCollectHooksErrorHooksSkippedOnSuccess(t *testing.T) {
	cfg := configfile.New("")
	cfg.Plugins = map[string]map[string]string{
		"nonexistent": {"error-hooks": "build"},
	}
	root := &cobra.Command{Use: "docker"}
	sub := &cobra.Command{Use: "build"}
	root.AddCommand(sub)

	// On success, error-hooks should not match, so the plugin
	// binary is never looked up and no results are returned.
	result := invokeAndCollectHooks(
		context.Background(), cfg, root, sub,
		"build", map[string]string{}, "",
	)
	assert.Check(t, is.Len(result, 0))
}

func TestInvokeAndCollectHooksNoPlugins(t *testing.T) {
	cfg := configfile.New("")
	root := &cobra.Command{Use: "docker"}
	sub := &cobra.Command{Use: "build"}
	root.AddCommand(sub)

	result := invokeAndCollectHooks(
		context.Background(), cfg, root, sub,
		"build", map[string]string{}, "some error",
	)
	assert.Check(t, is.Len(result, 0))
}

func TestInvokeAndCollectHooksCancelledContext(t *testing.T) {
	cfg := configfile.New("")
	cfg.Plugins = map[string]map[string]string{
		"test-plugin": {"hooks": "build"},
	}
	root := &cobra.Command{Use: "docker"}
	sub := &cobra.Command{Use: "build"}
	root.AddCommand(sub)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	result := invokeAndCollectHooks(
		ctx, cfg, root, sub,
		"build", map[string]string{}, "exit status 1",
	)
	assert.Check(t, is.Nil(result))
}
