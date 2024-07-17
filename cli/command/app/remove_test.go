package app

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/cli/internal/test"
	"gotest.tools/v3/assert"
)

func TestRunRemove(t *testing.T) {
	appBase := t.TempDir()
	t.Setenv("DOCKER_APP_BASE", appBase)

	binPath := filepath.Join(appBase, "bin")
	pkgPath := filepath.Join(appBase, "pkg")
	err := os.MkdirAll(binPath, 0o755)
	assert.NilError(t, err)
	err = os.MkdirAll(pkgPath, 0o755)
	assert.NilError(t, err)

	exist := func(p string) bool {
		_, err := os.Stat(p)
		return err == nil
	}

	create := func(p string) error {
		err := os.MkdirAll(filepath.Dir(p), 0o755)
		if err != nil {
			return err
		}
		f, err := os.Create(p)
		if err != nil {
			return err
		}
		err = f.Close()
		return err
	}

	createApp := func(name string, args []string) ([]string, error) {
		o := &AppOptions{
			commonOptions: commonOptions{
				_appBase: appBase,
				_args:    args,
			},
		}
		appPath, err := o.appPath()
		if err != nil {
			return nil, err
		}
		target := filepath.Join(appPath, name)
		link := filepath.Join(o.binPath(), name)
		err = create(target)
		if err != nil {
			return nil, err
		}
		err = os.Symlink(target, link)
		if err != nil {
			return nil, err
		}
		return []string{link, target}, nil
	}

	tests := []struct {
		name        string
		args        []string
		fakeInstall func([]string) []string
		expectErr   string
	}{
		{
			name: "one app", args: []string{"example.com/org/cool"},
			fakeInstall: func(args []string) []string {
				files, err := createApp("cool", args)
				assert.NilError(t, err)
				return files
			},
			expectErr: "",
		},
		{
			name: "a few apps", args: []string{"example.com/org/one", "example.com/org/two", "example.com/org/three@v1.2.3"},
			fakeInstall: func(args []string) []string {
				var files []string
				for _, a := range args {
					f, err := createApp(filepath.Base(a), []string{a})
					assert.NilError(t, err)
					files = append(files, f...)
				}
				return files
			}, expectErr: "",
		},
		{
			name: "none", args: []string{},
			fakeInstall: func(args []string) []string {
				return nil
			},
			expectErr: `"remove" requires at least 1 argument`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			files := tc.fakeInstall(tc.args)

			// make sure the files exist
			for _, f := range files {
				assert.Assert(t, exist(f))
			}

			cli := test.NewFakeCli(nil)
			cmd := NewRemoveCommand(cli)
			cmd.SetArgs(tc.args)
			cmd.SetOut(io.Discard)
			err := cmd.Execute()

			if tc.expectErr == "" {
				assert.NilError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.expectErr)
			}

			// assert the installed files are removed
			for _, f := range files {
				assert.Check(t, !exist(f))
			}
		})
	}
}
