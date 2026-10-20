package main

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/docker/cli/v29/cli-plugins/metadata"
	"github.com/docker/cli/v29/cli/command"
	"github.com/docker/cli/v29/cli/config"
	"github.com/docker/cli/v29/cli/config/configfile"
	contextdocker "github.com/docker/cli/v29/cli/context/docker"
	"github.com/docker/cli/v29/cli/context/store"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestCloudFlagHidden(t *testing.T) {
	dockerCli, err := command.NewDockerCli()
	assert.NilError(t, err)

	tcmd := newDockerCommand(dockerCli)
	tcmd.SetArgs([]string{"--cloud=team", "--help"})
	cmd, _, err := tcmd.HandleGlobalFlags()
	assert.NilError(t, err)

	name, err := cmd.Flags().GetString("cloud")
	assert.NilError(t, err)
	assert.Equal(t, name, "team")
	assert.Assert(t, cmd.Flags().Lookup("cloud").Hidden)
	assert.Assert(t, !strings.Contains(cmd.UsageString(), "--cloud"))
}

func TestCloudHelpRequest(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		help bool
	}{
		{name: "no command", help: true},
		{name: "global help", args: []string{"--help"}, help: true},
		{name: "global version", args: []string{"--version"}, help: true},
		{name: "help command", args: []string{"help", "run"}, help: true},
		{name: "completion request", args: []string{"__complete", "run", ""}, help: true},
		{name: "completion script", args: []string{"completion", "fish"}, help: true},
		{name: "builtin help", args: []string{"run", "--help"}, help: true},
		{name: "boolean help value", args: []string{"run", "--help=1"}, help: true},
		{name: "nested help", args: []string{"container", "run", "--help"}, help: true},
		{name: "plugin help", args: []string{"compose", "--help"}, help: true},
		{name: "command execution", args: []string{"ps"}},
		{name: "container help", args: []string{"run", "alpine", "--help"}},
		{name: "flag value", args: []string{"run", "--name", "--help", "alpine"}},
		{name: "help disabled", args: []string{"run", "--help=false", "alpine"}},
		{name: "build help", args: []string{"build", "--help"}, help: true},
		{name: "build buildkit-only flag", args: []string{"build", "--secret", "id=s,src=./s", "."}},
		{name: "image build buildkit-only flag", args: []string{"image", "build", "--push", "."}},
		{name: "builder build buildkit-only flag", args: []string{"builder", "build", "--ssh", "default", "."}},
		{name: "bake help", args: []string{"bake", "--help"}, help: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dockerCli, err := command.NewDockerCli()
			assert.NilError(t, err)
			tcmd := newDockerCommand(dockerCli)
			tcmd.SetArgs(append([]string{"--cloud"}, tc.args...))
			cmd, args, err := tcmd.HandleGlobalFlags()
			assert.NilError(t, err)

			help, err := cloudHelpRequest(cmd, args)
			assert.NilError(t, err)
			assert.Equal(t, help, tc.help)
		})
	}
}

// Flag errors are reported before cloud resolution, so they must not run the
// command's daemon feature checks against the engine selected before it.
func TestCloudFlagErrorSkipsDaemon(t *testing.T) {
	originalConfig := config.Dir()
	t.Cleanup(func() { config.SetDir(originalConfig) })
	config.SetDir(t.TempDir())

	var pings atomic.Int32
	daemon := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		pings.Add(1)
		w.Header().Set("Docker-Experimental", "false")
	}))
	defer daemon.Close()
	t.Setenv("DOCKER_HOST", "tcp://"+daemon.Listener.Addr().String())

	dockerCli, err := command.NewDockerCli()
	assert.NilError(t, err)
	tcmd := newDockerCommand(dockerCli)
	tcmd.SetArgs([]string{"--cloud", "checkpoint", "ls", "--unknown"})
	cmd, args, err := tcmd.HandleGlobalFlags()
	assert.NilError(t, err)
	assert.NilError(t, tcmd.Initialize())

	_, err = cloudHelpRequest(cmd, args)
	assert.Check(t, is.ErrorContains(err, "unknown flag: --unknown"))
	assert.Check(t, is.Equal(pings.Load(), int32(0)))
}

func TestCloudPluginArgs(t *testing.T) {
	for _, tc := range []struct {
		name    string
		args    []string
		context string
		want    []string
	}{
		{
			name:    "global flag only",
			args:    []string{"docker", "--cloud=foo", "compose", "up", "--cloud=child"},
			context: "cloud-foo",
			want:    []string{"docker", "--context=cloud-foo", "compose", "up", "--cloud=child"},
		},
		{
			name:    "flag value is not cloud flag",
			args:    []string{"docker", "--config", "--cloud", "--cloud", "-D", "compose", "up"},
			context: "cloud-default",
			want:    []string{"docker", "--context=cloud-default", "--config=--cloud", "--debug=true", "compose", "up"},
		},
		{
			name:    "delimiter preserved",
			args:    []string{"docker", "--cloud=foo", "--", "compose", "--", "--cloud"},
			context: "cloud-foo",
			want:    []string{"docker", "--context=cloud-foo", "--", "compose", "--", "--cloud"},
		},
		{
			name:    "repeated cloud flags",
			args:    []string{"docker", "--cloud=foo", "--cloud=bar", "compose", "up"},
			context: "cloud-bar",
			want:    []string{"docker", "--context=cloud-bar", "compose", "up"},
		},
		{
			name: "help without resolution",
			args: []string{"docker", "--cloud", "compose", "--help"},
			want: []string{"docker", "compose", "--help"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dockerCli, err := command.NewDockerCli()
			assert.NilError(t, err)
			tcmd := newDockerCommand(dockerCli)
			tcmd.SetArgs(tc.args[1:])
			cmd, _, err := tcmd.HandleGlobalFlags()
			assert.NilError(t, err)

			got, err := cloudPluginArgs(cmd, tc.args, tc.context)
			assert.NilError(t, err)
			assert.DeepEqual(t, got, tc.want)
		})
	}
}

// Exercise the real startup and subprocess boundary without a daemon or cloud
// service, including failures that must never dispatch the requested command.
func TestCloudResolution(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture plugins use shell scripts")
	}

	executable, err := os.Executable()
	assert.NilError(t, err)

	for _, tc := range []struct {
		name            string
		args            []string
		response        string
		exit            int
		wantName        string
		wantErr         string
		plugin          bool
		skip            bool
		provider        string
		noProvider      bool
		legacyProvider  bool
		invalidProvider bool
		wantHelp        bool
		command         []string
		visible         bool
	}{
		{
			name:     "default target",
			args:     []string{"--cloud"},
			wantName: "default",
		},
		{
			name:     "empty target",
			args:     []string{"--cloud="},
			wantName: "default",
		},
		{
			name:     "named target",
			args:     []string{"--cloud=team"},
			wantName: "team",
		},
		{
			name:     "plugin dispatch",
			args:     []string{"--cloud=team"},
			wantName: "team",
			plugin:   true,
		},
		{
			name:    "nonzero exit",
			args:    []string{"--cloud"},
			exit:    1,
			wantErr: "cloud resolver failed",
		},
		{
			name:     "malformed JSON",
			args:     []string{"--cloud"},
			response: "not JSON",
			wantErr:  "invalid cloud resolver response",
		},
		{
			name:     "multiple responses",
			args:     []string{"--cloud"},
			response: `{} {}`,
			wantErr:  "invalid cloud resolver response",
		},
		{
			name:     "missing field",
			args:     []string{"--cloud"},
			response: `{}`,
			wantErr:  "must return a non-default DOCKER_CONTEXT",
		},
		{
			name:     "local context",
			args:     []string{"--cloud"},
			response: `{"DOCKER_CONTEXT":"default"}`,
			wantErr:  "must return a non-default DOCKER_CONTEXT",
		},
		{
			name:     "missing context",
			args:     []string{"--cloud"},
			response: `{"DOCKER_CONTEXT":"missing"}`,
			wantErr:  `loading resolved context "missing"`,
		},
		{
			name:     "missing endpoint host",
			args:     []string{"--cloud"},
			response: `{"DOCKER_CONTEXT":"empty-host"}`,
			wantErr:  "has no Docker endpoint host",
		},
		{
			name:    "explicit context conflict",
			args:    []string{"--cloud", "--context=other"},
			wantErr: "conflicting options",
			skip:    true,
		},
		{
			name:    "explicit host conflict",
			args:    []string{"--cloud", "--host=tcp://localhost:2375"},
			wantErr: "conflicting options",
			skip:    true,
		},
		{
			name:    "invalid flag before help",
			args:    []string{"--cloud"},
			command: []string{"run", "--unknown", "--help"},
			wantErr: "unknown flag: --unknown",
			skip:    true,
		},
		{
			name:       "default provider",
			args:       []string{"--cloud"},
			noProvider: true,
			wantName:   "default",
		},
		{
			name:           "legacy default provider",
			args:           []string{"--cloud"},
			noProvider:     true,
			legacyProvider: true,
			wantErr:        `plugin "offload" does not support --cloud`,
			wantHelp:       true,
			skip:           true,
		},
		{
			name:            "invalid default provider",
			args:            []string{"--cloud"},
			noProvider:      true,
			invalidProvider: true,
			wantErr:         `invalid plugin "offload": plugin metadata does not define a vendor`,
			wantHelp:        true,
			skip:            true,
		},
		{
			name:            "invalid configured provider",
			args:            []string{"--cloud"},
			invalidProvider: true,
			wantErr:         `invalid plugin "foobar": plugin metadata does not define a vendor`,
			skip:            true,
		},
		{
			name:           "legacy configured provider",
			args:           []string{"--cloud"},
			legacyProvider: true,
			wantErr:        `plugin "foobar" does not support --cloud`,
			skip:           true,
		},
		{
			name:     "configured provider unavailable",
			args:     []string{"--cloud"},
			provider: "missing",
			wantErr:  `plugin "missing" unavailable`,
			skip:     true,
		},
		{
			name:     "provider cannot be a path",
			args:     []string{"--cloud"},
			provider: "../foobar",
			wantErr:  `plugin "../foobar" unavailable`,
			skip:     true,
		},
		{
			name:       "no cloud option",
			noProvider: true,
			skip:       true,
		},
		{
			name:       "help",
			args:       []string{"--cloud", "--help"},
			noProvider: true,
			skip:       true,
		},
		{
			name:       "version",
			args:       []string{"--cloud", "--version"},
			noProvider: true,
			skip:       true,
		},
		{
			name:       "help exposes supported provider",
			command:    []string{"--help"},
			noProvider: true,
			visible:    true,
			skip:       true,
		},
		{
			name:           "help hides legacy provider",
			command:        []string{"--help"},
			noProvider:     true,
			legacyProvider: true,
			skip:           true,
		},
		{
			name:     "help hides missing provider",
			command:  []string{"--help"},
			provider: "missing",
			skip:     true,
		},
		{
			name:       "completion exposes supported provider",
			command:    []string{"__complete", "--cl"},
			noProvider: true,
			visible:    true,
			skip:       true,
		},
		{
			name:           "completion hides legacy provider",
			command:        []string{"__complete", "--cl"},
			noProvider:     true,
			legacyProvider: true,
			skip:           true,
		},
		{
			name:     "completion hides missing provider",
			command:  []string{"__complete", "--cl"},
			provider: "missing",
			skip:     true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Explicit --cloud must override inherited endpoint selection.
			t.Setenv("DOCKER_CONTEXT", "inherited")
			t.Setenv("DOCKER_HOST", "tcp://127.0.0.1:1")
			t.Setenv("DOCKER_CLI_HOOKS", "false")
			t.Setenv("DOCKER_CONFIG", t.TempDir())

			configDir := t.TempDir()
			installedProvider := "offload"
			if !tc.noProvider {
				installedProvider = "foobar"
				cfg := configfile.New(filepath.Join(configDir, config.ConfigFileName))
				provider := cmp.Or(tc.provider, "foobar")
				cfg.Features = map[string]string{"cloud": provider}
				assert.NilError(t, cfg.Save())
			}

			pluginDir := filepath.Join(configDir, "cli-plugins")
			assert.NilError(t, os.MkdirAll(pluginDir, 0o755))

			response := cmp.Or(tc.response, `{"DOCKER_CONTEXT":"resolved"}`)

			providerMetadata := metadata.Metadata{SchemaVersion: "0.1.0", Vendor: "test"}
			if tc.invalidProvider {
				providerMetadata.Vendor = ""
			}
			if !tc.legacyProvider {
				providerMetadata.Features = map[string]any{cloudResolverFeature: true}
			}
			pluginMetadata, err := json.Marshal(providerMetadata)
			assert.NilError(t, err)

			script := fmt.Sprintf(`#!/bin/sh
if [ "$1" = docker-cli-plugin-metadata ]; then
    echo '%s'
    exit 0
fi
printf '%%s\n' "$@" >> "$DOCKER_CONFIG/resolver-args"
echo provisioning >&2
printf '%%s\n' '%s'
exit %d
`, pluginMetadata, response, tc.exit)
			assert.NilError(t, os.WriteFile(filepath.Join(pluginDir, "docker-"+installedProvider), []byte(script), 0o755))
			assert.NilError(t, os.WriteFile(filepath.Join(pluginDir, "docker-cloudtest"), []byte(`#!/bin/sh
if [ "$1" = docker-cli-plugin-metadata ]; then
    echo '{"SchemaVersion":"0.1.0","Vendor":"test"}'
    exit 0
fi
printf '%s\n' "$@" > "$CLOUD_TEST_ARGS"
`), 0o755))

			dispatchFile := filepath.Join(configDir, "dispatch-args")
			t.Setenv("CLOUD_TEST_ARGS", dispatchFile)

			contextStore := store.New(filepath.Join(configDir, "contexts"), command.DefaultContextStoreConfig())
			assert.NilError(t, contextStore.CreateOrUpdate(store.Metadata{
				Name:      "resolved",
				Endpoints: map[string]any{contextdocker.DockerEndpoint: contextdocker.EndpointMeta{Host: "tcp://127.0.0.1:1"}},
			}))
			assert.NilError(t, contextStore.CreateOrUpdate(store.Metadata{
				Name:      "empty-host",
				Endpoints: map[string]any{contextdocker.DockerEndpoint: contextdocker.EndpointMeta{}},
			}))

			args := append([]string{"docker", "--config=" + configDir}, tc.args...)
			switch {
			case tc.command != nil:
				args = append(args, tc.command...)
			case tc.plugin || tc.wantErr != "":
				args = append(args, "cloudtest", "up", "--cloud=child")
			default:
				args = append(args, "context", "show")
			}

			payload, err := json.Marshal(args)
			assert.NilError(t, err)

			var stdout, stderr bytes.Buffer
			cmd := exec.CommandContext(t.Context(), executable, "-test.run=^TestCloudCommandProcess$")
			cmd.Env = append(os.Environ(), "CLOUD_TEST_COMMAND="+string(payload))
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err = cmd.Run()
			if tc.wantErr != "" {
				assert.Assert(t, err != nil)
				assert.Check(t, is.Contains(stderr.String(), tc.wantErr))
				assert.Equal(t, strings.Contains(stderr.String(), cloudPluginHelp), tc.wantHelp, stderr.String())
				_, statErr := os.Stat(dispatchFile)
				assert.Assert(t, os.IsNotExist(statErr))
				assert.Equal(t, stdout.String(), "")
			} else {
				assert.NilError(t, err, stderr.String())
				if tc.command != nil {
					assert.Equal(t, strings.Contains(stdout.String(), "--cloud"), tc.visible, stdout.String())
				}
				if !tc.skip {
					if tc.plugin {
						args, err := os.ReadFile(dispatchFile)
						assert.NilError(t, err)
						assert.Equal(t, string(args), "--context=resolved\n--config="+configDir+"\ncloudtest\nup\n--cloud=child\n")
					} else {
						assert.Equal(t, stdout.String(), "resolved\n")
					}
				}
			}

			invocation, err := os.ReadFile(filepath.Join(configDir, "resolver-args"))
			if tc.skip {
				assert.Assert(t, os.IsNotExist(err))
			} else {
				assert.NilError(t, err)
				assert.Check(t, is.Contains(stderr.String(), "provisioning"))
				name := cmp.Or(tc.wantName, "default")
				assert.Equal(t, string(invocation), "--config="+configDir+"\n"+installedProvider+"\n__resolve-context\n--\n"+name+"\n")
			}
		})
	}
}

func TestCloudResolverInput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture plugins use shell scripts and pseudo-terminals")
	}

	executable, err := os.Executable()
	assert.NilError(t, err)

	for _, tc := range []struct {
		name        string
		stdinType   string
		terminalErr bool
		prompt      bool
	}{
		{name: "interactive", stdinType: "terminal", terminalErr: true, prompt: true},
		{name: "piped stdin", stdinType: "pipe", terminalErr: true},
		{name: "redirected stdin", stdinType: "file", terminalErr: true},
		{name: "redirected stderr", stdinType: "terminal"},
		{name: "noninteractive", stdinType: "pipe"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DOCKER_CLI_HOOKS", "false")
			configDir := t.TempDir()
			pluginDir := filepath.Join(configDir, "cli-plugins")
			assert.NilError(t, os.MkdirAll(pluginDir, 0o755))
			assert.NilError(t, os.WriteFile(filepath.Join(pluginDir, "docker-offload"), []byte(`#!/bin/sh
if [ "$1" = docker-cli-plugin-metadata ]; then
    echo '{"SchemaVersion":"0.1.0","Vendor":"test","Features":{"cloud-context-resolver":true}}'
    exit 0
fi
if [ -t 0 ]; then
    [ -t 2 ] || exit 1
    printf 'Continue? ' >&2
    IFS= read -r reply || exit 1
    printf '%s\n' "$reply" > "$DOCKER_CONFIG/resolver-input"
elif IFS= read -r unexpected; then
    echo 'resolver consumed command input' >&2
    exit 1
fi
echo '{"DOCKER_CONTEXT":"resolved"}'
`), 0o755))
			assert.NilError(t, os.WriteFile(filepath.Join(pluginDir, "docker-cloudtest"), []byte(`#!/bin/sh
if [ "$1" = docker-cli-plugin-metadata ]; then
    echo '{"SchemaVersion":"0.1.0","Vendor":"test"}'
    exit 0
fi
IFS= read -r input || exit 1
printf '%s\n' "$input"
`), 0o755))

			contextStore := store.New(filepath.Join(configDir, "contexts"), command.DefaultContextStoreConfig())
			assert.NilError(t, contextStore.CreateOrUpdate(store.Metadata{
				Name:      "resolved",
				Endpoints: map[string]any{contextdocker.DockerEndpoint: contextdocker.EndpointMeta{Host: "tcp://127.0.0.1:1"}},
			}))

			terminal, tty, err := pty.Open()
			assert.NilError(t, err)
			t.Cleanup(func() {
				_ = tty.Close()
				_ = terminal.Close()
			})

			const commandInput = "input for the original command\n"
			var stdin *os.File
			switch tc.stdinType {
			case "terminal":
				stdin = tty
				input := commandInput
				if tc.prompt {
					input = "yes\n" + input
				}
				_, err = terminal.WriteString(input)
				assert.NilError(t, err)
			case "pipe":
				var writer *os.File
				stdin, writer, err = os.Pipe()
				assert.NilError(t, err)
				t.Cleanup(func() { _ = stdin.Close() })
				t.Cleanup(func() { _ = writer.Close() })
				_, err = writer.WriteString(commandInput)
				assert.NilError(t, err)
				assert.NilError(t, writer.Close())
			case "file":
				inputPath := filepath.Join(configDir, "command-input")
				assert.NilError(t, os.WriteFile(inputPath, []byte(commandInput), 0o600))
				stdin, err = os.Open(inputPath)
				assert.NilError(t, err)
				t.Cleanup(func() { _ = stdin.Close() })
			}

			payload, err := json.Marshal([]string{"docker", "--config=" + configDir, "--cloud", "cloudtest"})
			assert.NilError(t, err)
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			var stdout, stderr bytes.Buffer
			cmd := exec.CommandContext(ctx, executable, "-test.run=^TestCloudCommandProcess$")
			cmd.Env = append(os.Environ(), "CLOUD_TEST_COMMAND="+string(payload))
			cmd.Stdin = stdin
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			cmd.WaitDelay = time.Second
			if tc.terminalErr {
				cmd.Stderr = tty
			}
			assert.NilError(t, cmd.Run(), stderr.String())
			assert.Equal(t, stdout.String(), commandInput)

			reply, err := os.ReadFile(filepath.Join(configDir, "resolver-input"))
			if tc.prompt {
				assert.NilError(t, err)
				assert.Equal(t, string(reply), "yes\n")
			} else {
				assert.Assert(t, os.IsNotExist(err))
			}
		})
	}
}

// Run startup in a separate process because it installs process-wide signal
// handlers that intentionally outlive an individual command.
func TestCloudCommandProcess(t *testing.T) {
	payload := os.Getenv("CLOUD_TEST_COMMAND")
	if payload == "" {
		return
	}

	assert.NilError(t, json.Unmarshal([]byte(payload), &os.Args))

	err := dockerMain(context.Background())
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
	}
	if err == nil {
		os.Exit(0)
	}

	os.Exit(getExitCode(err))
}

func TestCloudResolutionCancellation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture plugin uses a shell script")
	}

	originalConfig := config.Dir()
	t.Cleanup(func() { config.SetDir(originalConfig) })
	config.SetDir(t.TempDir())

	cfg := configfile.New(filepath.Join(config.Dir(), config.ConfigFileName))
	cfg.Features = map[string]string{"cloud": "foobar"}
	assert.NilError(t, cfg.Save())

	pluginDir := filepath.Join(config.Dir(), "cli-plugins")
	assert.NilError(t, os.MkdirAll(pluginDir, 0o755))
	assert.NilError(t, os.WriteFile(filepath.Join(pluginDir, "docker-foobar"), []byte(`#!/bin/sh
if [ "$1" = docker-cli-plugin-metadata ]; then
    echo '{"SchemaVersion":"0.1.0","Vendor":"test","Features":{"cloud-context-resolver":true}}'
    exit 0
fi
exec sleep 30
`), 0o755))

	dockerCli, err := command.NewDockerCli()
	assert.NilError(t, err)
	tcmd := newDockerCommand(dockerCli)
	tcmd.SetArgs(strings.Fields("--cloud context show"))
	cmd, _, err := tcmd.HandleGlobalFlags()
	assert.NilError(t, err)

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	_, err = resolveCloudContext(ctx, dockerCli, cmd, "default")
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}
