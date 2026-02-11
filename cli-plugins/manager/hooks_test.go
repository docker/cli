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
		commandString string
		pluginConfig  map[string]string
		expectedMatch string
		expectedOk    bool
	}{
		{
			commandString: "image ls",
			pluginConfig: map[string]string{
				"hooks": "image",
			},
			expectedMatch: "image",
			expectedOk:    true,
		},
		{
			commandString: "context ls",
			pluginConfig: map[string]string{
				"hooks": "build",
			},
			expectedMatch: "",
			expectedOk:    false,
		},
		{
			commandString: "context ls",
			pluginConfig: map[string]string{
				"hooks": "context ls",
			},
			expectedMatch: "context ls",
			expectedOk:    true,
		},
		{
			commandString: "image ls",
			pluginConfig: map[string]string{
				"hooks": "image ls,image",
			},
			expectedMatch: "image ls",
			expectedOk:    true,
		},
		{
			commandString: "image ls",
			pluginConfig: map[string]string{
				"hooks": "",
			},
			expectedMatch: "",
			expectedOk:    false,
		},
		{
			commandString: "image inspect",
			pluginConfig: map[string]string{
				"hooks": "image i",
			},
			expectedMatch: "",
			expectedOk:    false,
		},
		{
			commandString: "image inspect",
			pluginConfig: map[string]string{
				"hooks": "image",
			},
			expectedMatch: "image",
			expectedOk:    true,
		},
	}

	for _, tc := range testCases {
		match, ok := pluginMatch(tc.pluginConfig, tc.commandString)
		assert.Equal(t, ok, tc.expectedOk)
		assert.Equal(t, match, tc.expectedMatch)
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

func TestInvokeAndCollectHooksForwardsErrorMessage(t *testing.T) {
	cfg := configfile.New("")
	cfg.Plugins = map[string]map[string]string{
		"nonexistent": {"hooks": "build"},
	}
	root := &cobra.Command{Use: "docker"}
	sub := &cobra.Command{Use: "build"}
	root.AddCommand(sub)

	// Plugin binary doesn't exist — invokeAndCollectHooks skips it
	// gracefully and returns empty. Verifies the error message path
	// doesn't cause issues when forwarded through the call chain.
	result := invokeAndCollectHooks(
		context.Background(), cfg, root, sub,
		"build", map[string]string{}, "exit status 1",
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
