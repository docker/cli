package app

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

// spawn runs the specified command
func spawn(bin string, args []string, envMap map[string]string, detach bool) error {
	toEnv := func() []string {
		var env []string
		for k, v := range envMap {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		return env
	}

	cmd := exec.Command(bin, args...)
	cmd.Env = append(os.Environ(), toEnv()...)
	cmd.Dir = filepath.Dir(bin)
	if detach {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
		}
		err := cmd.Start()
		if err != nil {
			return err
		}
		return nil
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		err := cmd.Start()
		if err != nil {
			return err
		}
		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()
		select {
		case sig := <-sigs:
			if cmd.Process != nil {
				cmd.Process.Signal(sig)
			}
			return fmt.Errorf("signal received: %v", sig)
		case err := <-done:
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// oneChild checks if the directory contains only one single file.
// if true, return the file path
func oneChild(dir string) (string, error) {
	dirs, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	var fp string
	cnt := 0
	for _, v := range dirs {
		if !v.IsDir() {
			cnt++
			if cnt > 1 {
				break
			}
			fp = filepath.Join(dir, v.Name())
		}
	}
	if cnt != 1 {
		return "", nil
	}

	ap, err := filepath.Abs(fp)
	if err != nil {
		return "", err
	}
	return ap, nil
}

// locateFile searches for the filename in a given directory
// if found, return its file path
func locateFile(dir, name string) (string, error) {
	dirs, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	for _, entry := range dirs {
		if !entry.IsDir() && entry.Name() == name {
			fp := filepath.Join(dir, entry.Name())
			ap, err := filepath.Abs(fp)
			if err != nil {
				return "", err
			}
			return ap, nil
		}
	}
	return "", nil
}

// parseURL normalizes the given string as URL
// currently supported schemes: file, http, https, git
func parseURL(s string) (*url.URL, error) {
	if !strings.Contains(s, "://") {
		ap, err := filepath.Abs(s)
		if err != nil {
			return nil, err
		}
		s = "file://" + ap
	}

	parsed, err := url.Parse(s)
	if err != nil {
		return nil, err
	}
	switch parsed.Scheme {
	case "file", "http", "https", "git":
		return parsed, nil
	default:
		return nil, fmt.Errorf("not supported: %s", s)
	}
}

// isSymlinkOK checks if it is ok to create a symlink to the target
// it is ok if the path does not exist
// or if the path is a symlink that points to the same target.
// it is considered the same target if the symlink is identical up to
// the first @ or # sign, i.e. they are the same app but different versions.
func isSymlinkOK(path, target string) (bool, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}

	if fi.Mode()&os.ModeSymlink == 0 {
		return false, fmt.Errorf("another app/version exists: %s", path)
	}

	link, err := os.Readlink(path)
	if err != nil {
		return false, err
	}

	pkg := func(s string) string {
		s = filepath.Dir(s)
		s = strings.Split(s, "#")[0]
		return strings.Split(s, "@")[0]
	}

	return pkg(link) == pkg(target), nil
}

// splitAtDashDash splits a string array into two parts
// at the first double dash "--"
func splitAtDashDash(arr []string) ([]string, []string) {
	for i, v := range arr {
		if v == "--" {
			if i+1 < len(arr) {
				return arr[:i], arr[i+1:]
			}
			return arr[:i], []string{}
		}
	}
	return arr, []string{}
}

// findLinks returns a list of symlinks in the given directory
func findLinks(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var links []string
	for _, v := range entries {
		p := filepath.Join(dir, v.Name())
		if v.Type()&os.ModeSymlink != 0 {
			links = append(links, p)
		}
	}

	return links, nil
}

// removeEmptyPath removes the dir and all its ancestors if empty
// until it reaches the root
func removeEmptyPath(root, dir string) error {
	root = filepath.Clean(root)
	dir = filepath.Clean(dir)

	if !strings.HasPrefix(dir, root) {
		return nil
	}

	var rm func(string) error
	rm = func(p string) error {
		if p == root {
			return nil
		}
		if err := os.Remove(p); err != nil {
			if !os.IsNotExist(err) {
				return err
			}
		}
		parent := filepath.Dir(p)
		return rm(parent)
	}

	return rm(filepath.Dir(dir))
}

// shortenPath removes home directory from path to make it shorter
func shortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return strings.ReplaceAll(path, home+"/", "_")
}

// locateDir walks back the path and looks for directory with the given name.
// If found, it returns the directory; otherwise an empty string.
func locateDir(path, name string) (string, error) {
	check := func(dir string) (bool, string) {
		if filepath.Base(dir) == name {
			return true, dir
		}
		child := filepath.Join(dir, name)
		info, err := os.Stat(child)
		if err != nil {
			return false, ""
		}
		return info.IsDir(), child
	}

	dir, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	for {
		if found, d := check(dir); found {
			return d, nil
		}

		parent := filepath.Dir(dir)
		if parent == "/" || parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("not found: %s", name)
}
