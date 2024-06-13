//go:build unix

package completion

import (
	"context"
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

//go:embed testdata
var fixtures embed.FS

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

type fakeGenerate struct{}

var _ generateCompletions = (*fakeGenerate)(nil)

func (f *fakeGenerate) GenBashCompletionV2(w io.Writer, includeDesc bool) error {
	completions, err := fixtures.ReadFile("testdata/docker.bash")
	if err != nil {
		return err
	}
	_, err = w.Write(completions)
	return err
}

func (f *fakeGenerate) GenZshCompletion(w io.Writer) error {
	completions, err := fixtures.ReadFile("testdata/docker.zsh")
	if err != nil {
		return err
	}
	_, err = w.Write(completions)
	return err
}

func (f *fakeGenerate) GenFishCompletion(w io.Writer, includeDesc bool) error {
	completions, err := fixtures.ReadFile("testdata/docker.fish")
	if err != nil {
		return err
	}
	_, err = w.Write(completions)
	return err
}

func newFakeGenerate() generateCompletions {
	return &fakeGenerate{}
}

func TestDockerCompletion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	t.Run("zsh completion", func(t *testing.T) {
		t.Setenv("SHELL", "/bin/zsh")
		ds, err := NewShellCompletionSetup("", newFakeGenerate())
		assert.NilError(t, err)

		completions, err := ds.GetCompletionScript(ctx)
		assert.NilError(t, err, "expected docker completions to not error, got %s", err)
		assert.Check(t, len(completions) > 0, "expected docker completions to not be empty")

		expected, err := fixtures.ReadFile("testdata/docker.zsh")
		assert.NilError(t, err)

		assert.Equal(t, string(expected), string(completions), "docker.zsh fixture did not match docker completion output")
	})

	t.Run("bash completion", func(t *testing.T) {
		t.Setenv("SHELL", "/bin/bash")
		ds, err := NewShellCompletionSetup("", newFakeGenerate())
		assert.NilError(t, err)

		completions, err := ds.GetCompletionScript(ctx)
		assert.NilError(t, err)
		assert.Check(t, len(completions) > 0, "expected docker completions to not be empty")

		expected, err := fixtures.ReadFile("testdata/docker.bash")
		assert.NilError(t, err)

		assert.Equal(t, string(expected), string(completions), "docker.bash fixtures did not match docker completion output")
	})

	t.Run("fish completion", func(t *testing.T) {
		t.Setenv("SHELL", "/bin/fish")
		ds, err := NewShellCompletionSetup("", newFakeGenerate())
		assert.NilError(t, err)

		completions, err := ds.GetCompletionScript(ctx)
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

			s, shellRaw, err := shellFromEnv()
			if tc.expectedErr != "" {
				assert.Check(t, is.ErrorContains(err, tc.expectedErr))
			} else {
				assert.Equal(t, tc.expected, s)
				assert.Equal(t, string(tc.expected), shellRaw)
			}
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
		ds, err := NewShellCompletionSetup(tmphome, newFakeGenerate())
		assert.NilError(t, err)
		return ds.(*unixShellSetup)
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

				ds, err := newUnixShellSetup(tmphome, newFakeGenerate())
				assert.NilError(t, err)

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

				ds, err := newUnixShellSetup(tmphome, newFakeGenerate())
				assert.NilError(t, err)

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

			t.Setenv("SHELL", string(tc.shell))
			ds := tc.setupFunc(t)
			assert.NilError(t, ds.InstallCompletions(ctx))
			tc.assertFunc(t, ds)
		})
	}
}

func TestGetCompletionDir(t *testing.T) {
	t.Run("standard shells", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		tmphome := t.TempDir()

		for _, tc := range []struct {
			shell    string
			expected string
		}{
			{"/bin/bash", filepath.Join(tmphome, bashCompletionDir)},
			{"/bin/zsh", filepath.Join(tmphome, zshCompletionDir)},
			{"/bin/fish", filepath.Join(tmphome, fishCompletionDir)},
		} {
			t.Setenv("SHELL", tc.shell)
			ds, err := NewShellCompletionSetup(tmphome, newFakeGenerate())
			assert.NilError(t, err)
			assert.Equal(t, tc.expected, ds.GetCompletionDir(ctx))
		}
	})

	t.Run("oh-my-zsh", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		tmphome := t.TempDir()
		ohMyZshTmpDir := filepath.Join(tmphome, ".oh-my-zsh")
		assert.NilError(t, os.MkdirAll(ohMyZshTmpDir, filePerm))
		t.Setenv("SHELL", "/bin/zsh")
		t.Setenv("ZSH", ohMyZshTmpDir)
		ds, err := NewShellCompletionSetup(tmphome, newFakeGenerate())
		assert.NilError(t, err)
		assert.Equal(t, filepath.Join(ohMyZshTmpDir, "completions"), ds.GetCompletionDir(ctx))
	})
}
