package completion

import (
	"context"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

type testFuncs string

const (
	testDockerCompletions testFuncs = "TestDockerCompletions"
)

//go:embed testdata
var fixtures embed.FS

// fakeExecCommand is a helper function that hooks
// the current test binary into an os/exec cmd.Run() call
// allowing us to mock out third party dependencies called through os/exec.
//
// testBinary is the current test binary that is running, can be accessed through os.Args[0]
// funcName is the name of the function you want to run as a sub-process of the current test binary
//
// The call path is as follows:
// - Register the function you want to run through TestMain
// - Call the cmd.Run() function from the returned exec.Cmd
// - TestMain will execute the function as a sub-process of the current test binary
func fakeExecCommand(t *testing.T, testBinary string, funcName testFuncs) func(ctx context.Context, command string, args ...string) *exec.Cmd {
	t.Helper()
	return func(ctx context.Context, command string, args ...string) *exec.Cmd {
		cmd := exec.Command(testBinary, append([]string{command}, args...)...)
		cmd.Env = append(os.Environ(), "TEST_MAIN_FUNC="+string(funcName))
		return cmd
	}
}

// TestMain is setup here to act as a dispatcher
// for functions hooked into the test binary through
// fakeExecCommand.
func TestMain(m *testing.M) {
	switch testFuncs(os.Getenv("TEST_MAIN_FUNC")) {
	case testDockerCompletions:
		FakeDockerCompletionsProcess()
	default:
		os.Exit(m.Run())
	}
}

// this is a test function that will only be run when
// fakeExecCommand is hooked into a cmd.Run()/cmd.Output call
// with the funcName as "TestDockerCompletions"
// TestMain executes this function as a sub-process of the current
// test binary.
func FakeDockerCompletionsProcess() {
	s := supportedCompletionShell(os.Args[3])
	if s == "" {
		panic("shell not provided")
	}

	completions, err := fixtures.ReadFile("testdata/docker." + string(s))
	if err != nil {
		panic(err)
	}

	_, err = fmt.Fprint(os.Stdout, string(completions))
	if err != nil {
		panic(err)
	}

	os.Exit(0)
}

func TestDockerCompletion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	ds := NewUnixShellSetup("", "docker").(*unixShellSetup)
	ds.command = fakeExecCommand(t, os.Args[0], testDockerCompletions)

	t.Run("zsh completion", func(t *testing.T) {
		completions, err := ds.DockerCompletion(ctx, zsh)
		assert.NilError(t, err, "expected docker completions to not error, got %s", err)
		assert.Check(t, len(completions) > 0, "expected docker completions to not be empty")

		expected, err := fixtures.ReadFile("testdata/docker.zsh")
		assert.NilError(t, err)

		assert.Equal(t, string(expected), string(completions), "docker.zsh fixture did not match docker completion output")
	})

	t.Run("bash completion", func(t *testing.T) {
		completions, err := ds.DockerCompletion(ctx, bash)
		assert.NilError(t, err)
		assert.Check(t, len(completions) > 0, "expected docker completions to not be empty")

		expected, err := fixtures.ReadFile("testdata/docker.bash")
		assert.NilError(t, err)

		assert.Equal(t, string(expected), string(completions), "docker.bash fixtures did not match docker completion output")
	})

	t.Run("fish completion", func(t *testing.T) {
		completions, err := ds.DockerCompletion(ctx, fish)
		assert.NilError(t, err)
		assert.Check(t, len(completions) > 0, "expected docker completions to not be empty")

		expected, err := fixtures.ReadFile("testdata/docker.fish")
		assert.NilError(t, err)

		assert.Equal(t, string(expected), string(completions), "docker.fish fixtures did not match docker completion output")
	})
}

func TestUnixDefaultShell(t *testing.T) {
	for _, tc := range []struct {
		desc        string
		path        string
		expected    supportedCompletionShell
		expectedErr string
	}{
		{
			desc:        "bash",
			path:        "/bin/bash",
			expected:    bash,
			expectedErr: "",
		},
		{
			desc:        "zsh",
			path:        "/bin/zsh",
			expected:    zsh,
			expectedErr: "",
		},
		{
			desc:        "fish",
			path:        "/bin/fish",
			expected:    fish,
			expectedErr: "",
		},
		{
			desc:        "homebrew bash",
			path:        "/opt/homebrew/bin/bash",
			expected:    bash,
			expectedErr: "",
		},
		{
			desc:        "homebrew zsh",
			path:        "/opt/homebrew/bin/zsh",
			expected:    zsh,
			expectedErr: "",
		},
		{
			desc:        "homebrew fish",
			path:        "/opt/homebrew/bin/fish",
			expected:    fish,
			expectedErr: "",
		},
		{
			desc:        "unsupported shell",
			path:        "/bin/unsupported",
			expected:    "",
			expectedErr: "unsupported shell",
		},
		{
			desc:        "empty shell",
			path:        "",
			expected:    "",
			expectedErr: "SHELL environment variable not set",
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			t.Setenv("SHELL", tc.path)

			s, err := unixDefaultShell()
			if tc.expectedErr != "" {
				assert.Check(t, is.ErrorContains(err, tc.expectedErr))
			}
			assert.Equal(t, tc.expected, s)
		})
	}
}

func TestInstallCompletions(t *testing.T) {

	zshSetup := func(t *testing.T, ds *unixShellSetup) *os.File {
		t.Helper()
		zshrcFile, err := os.OpenFile(ds.zshrc, os.O_RDWR|os.O_CREATE, filePerm)
		assert.NilError(t, err)

		t.Cleanup(func() {
			zshrcFile.Close()
		})

		_, err = os.Stat(ds.zshrc)
		assert.NilError(t, err, "expected zshrc file to exist")
		return zshrcFile
	}

	hasZshCompletions := func(t *testing.T, ds *unixShellSetup) {
		t.Helper()

		zshrcContent, err := os.ReadFile(ds.zshrc)
		assert.NilError(t, err)
		assert.Check(t, is.Contains(string(zshrcContent), fmt.Sprintf("fpath=(%s $fpath)", ds.zshCompletionDir)))

		_, err = os.Stat(filepath.Join(ds.zshCompletionDir, zsh.FileName()))
		assert.NilError(t, err, "expected zsh completions directory to exist")

		completions, err := os.ReadFile(filepath.Join(ds.zshCompletionDir, zsh.FileName()))
		assert.NilError(t, err)

		zshFixture, err := fixtures.ReadFile("testdata/docker." + string(zsh))
		assert.NilError(t, err)
		assert.Equal(t, string(zshFixture), string(completions))
	}

	setup := func(t *testing.T) *unixShellSetup {
		t.Helper()

		tmphome := t.TempDir()
		ds := NewUnixShellSetup(tmphome, "docker").(*unixShellSetup)
		ds.command = fakeExecCommand(t, os.Args[0], testDockerCompletions)
		return ds
	}

	testcases := []struct {
		shell      supportedCompletionShell
		desc       string
		setupFunc  func(t *testing.T) *unixShellSetup
		assertFunc func(t *testing.T, ds *unixShellSetup)
	}{
		{
			shell: zsh,
			desc:  "zsh completions",
			setupFunc: func(t *testing.T) *unixShellSetup {
				t.Helper()
				ds := setup(t)
				zshSetup(t, ds)
				return ds
			},
			assertFunc: hasZshCompletions,
		},
		{
			shell: zsh,
			desc:  "zsh completions with ZDOTDIR",
			setupFunc: func(t *testing.T) *unixShellSetup {
				t.Helper()
				zdotdir := filepath.Join(t.TempDir(), "zdotdir")
				assert.NilError(t, os.MkdirAll(zdotdir, filePerm))
				t.Setenv("ZDOTDIR", zdotdir)

				ds := setup(t)
				zshSetup(t, ds)
				return ds
			},
			assertFunc: func(t *testing.T, ds *unixShellSetup) {
				t.Helper()
				hasZshCompletions(t, ds)
				assert.Check(t, is.Contains(os.Getenv("ZDOTDIR"), "zdotdir"))
				assert.Equal(t, ds.zshrc, filepath.Join(os.Getenv("ZDOTDIR"), ".zshrc"))
			},
		},
		{
			shell: zsh,
			desc:  "existing fpath in zshrc",
			setupFunc: func(t *testing.T) *unixShellSetup {
				t.Helper()
				ds := setup(t)
				zshrcFile := zshSetup(t, ds)

				_, err := fmt.Fprintf(zshrcFile, "fpath=(%s $fpath)", ds.zshCompletionDir)
				assert.NilError(t, err)

				return ds
			},
			assertFunc: func(t *testing.T, ds *unixShellSetup) {
				t.Helper()
				hasZshCompletions(t, ds)
				zshrcFile, err := os.ReadFile(ds.zshrc)
				assert.NilError(t, err)
				assert.Equal(t, 1, strings.Count(string(zshrcFile), fmt.Sprintf("fpath=(%s $fpath)", ds.zshCompletionDir)))
			},
		},
		{
			shell:     bash,
			desc:      "bash completions",
			setupFunc: setup,
			assertFunc: func(t *testing.T, ds *unixShellSetup) {
				t.Helper()
				_, err := os.Stat(filepath.Join(ds.bashCompletionDir, bash.FileName()))
				assert.NilError(t, err)

				completions, err := os.ReadFile(filepath.Join(ds.bashCompletionDir, bash.FileName()))
				assert.NilError(t, err)

				bashFixture, err := fixtures.ReadFile("testdata/docker." + string(bash))
				assert.NilError(t, err)
				assert.Equal(t, string(bashFixture), string(completions))
			},
		},
		{
			shell:     fish,
			desc:      "fish completions",
			setupFunc: setup,
			assertFunc: func(t *testing.T, ds *unixShellSetup) {
				t.Helper()
				_, err := os.Stat(filepath.Join(ds.fishCompletionDir, fish.FileName()))
				assert.NilError(t, err)

				completions, err := os.ReadFile(filepath.Join(ds.fishCompletionDir, fish.FileName()))
				assert.NilError(t, err)

				fishFixture, err := fixtures.ReadFile("testdata/docker." + string(fish))
				assert.NilError(t, err)
				assert.Equal(t, string(fishFixture), string(completions))
			},
		},
		{
			shell: zsh,
			desc:  "zsh with oh-my-zsh",
			setupFunc: func(t *testing.T) *unixShellSetup {
				t.Helper()
				tmphome := t.TempDir()
				ohmyzsh := filepath.Join(tmphome, ".oh-my-zsh")
				assert.NilError(t, os.MkdirAll(ohmyzsh, filePerm))
				t.Setenv("ZSH", ohmyzsh)

				ds := NewUnixShellSetup(tmphome, "docker").(*unixShellSetup)
				ds.command = fakeExecCommand(t, os.Args[0], testDockerCompletions)

				zshSetup(t, ds)
				return ds
			},
			assertFunc: func(t *testing.T, ds *unixShellSetup) {
				t.Helper()
				assert.Equal(t, filepath.Join(ds.homeDirectory, ".oh-my-zsh/completions"), ds.zshCompletionDir)
				_, err := os.Stat(filepath.Join(ds.zshCompletionDir, zsh.FileName()))
				assert.NilError(t, err)

				completions, err := os.ReadFile(filepath.Join(ds.zshCompletionDir, zsh.FileName()))
				assert.NilError(t, err)

				zshFixture, err := fixtures.ReadFile("testdata/docker." + string(zsh))
				assert.NilError(t, err)
				assert.Equal(t, string(zshFixture), string(completions))
			},
		},
		{
			shell: zsh,
			desc:  "should fallback to zsh when oh-my-zsh directory does not exist",
			setupFunc: func(t *testing.T) *unixShellSetup {
				t.Helper()
				tmphome := t.TempDir()
				t.Setenv("ZSH", filepath.Join(tmphome, ".oh-my-zsh"))

				ds := NewUnixShellSetup(tmphome, "docker").(*unixShellSetup)
				ds.command = fakeExecCommand(t, os.Args[0], testDockerCompletions)

				zshSetup(t, ds)
				return ds
			},
			assertFunc: func(t *testing.T, ds *unixShellSetup) {
				t.Helper()
				assert.Check(t, !strings.Contains(ds.zshCompletionDir, ".oh-my-zsh"))
				hasZshCompletions(t, ds)
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)

			ds := tc.setupFunc(t)
			assert.NilError(t, ds.InstallCompletions(ctx, tc.shell))
			tc.assertFunc(t, ds)
		})
	}
}

func TestGetCompletionDir(t *testing.T) {

	t.Run("standard shells", func(t *testing.T) {
		tmphome := t.TempDir()
		ds := NewUnixShellSetup(tmphome, "docker")

		assert.Equal(t, filepath.Join(tmphome, zshCompletionDir), ds.GetCompletionDir(zsh))
		assert.Equal(t, filepath.Join(tmphome, fishCompletionDir), ds.GetCompletionDir(fish))
		assert.Equal(t, filepath.Join(tmphome, bashCompletionDir), ds.GetCompletionDir(bash))
		assert.Equal(t, "", ds.GetCompletionDir(supportedCompletionShell("unsupported")))
	})

	t.Run("oh-my-zsh", func(t *testing.T) {
		tmphome := t.TempDir()
		ohMyZshTmpDir := filepath.Join(tmphome, ".oh-my-zsh")
		assert.NilError(t, os.MkdirAll(ohMyZshTmpDir, filePerm))
		t.Setenv("ZSH", ohMyZshTmpDir)
		ds := NewUnixShellSetup(tmphome, "docker")
		assert.Equal(t, filepath.Join(ohMyZshTmpDir, "completions"), ds.GetCompletionDir(zsh))
	})

}
