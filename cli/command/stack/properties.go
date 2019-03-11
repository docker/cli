package stack

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/cli/opts"
	"github.com/docker/stacks/pkg/types"
)

func substituteProperties(input *types.ComposeInput, workingDir string) error {
	// Load up an .env file if it exists
	overrides := map[string]string{}
	fileOverrides, err := opts.ParseEnvFile(filepath.Join(workingDir, ".env"))
	if err == nil {
		for _, line := range fileOverrides {
			e := strings.SplitN(line, "=", 2)
			if len(e) != 2 {
				return fmt.Errorf("malformed env file %s - %s", filepath.Join(workingDir, ".env"), line)
			}
			overrides[e[0]] = e[1]
		}
	}

	missing := []string{}
	for i, tmpl := range input.ComposeFiles {
		input.ComposeFiles[i] = os.Expand(tmpl, func(key string) string {
			if key == "$" {
				return "$$"
			}
			var name string
			var defaultValue string
			var errString string
			matched := false
			for _, sep := range []string{":-", "-"} {
				name, defaultValue = partition(key, sep)
				if defaultValue != "" {
					matched = true
					break
				}
			}
			// Check for mandatory fields
			if !matched {
				for _, sep := range []string{":?", "?"} {
					name, errString = partition(key, sep)
					if errString != "" {
						matched = true
						break
					}
				}
			}
			if !matched {
				name = key
			}
			if value, found := overrides[name]; found {
				return value
			}
			if value, found := os.LookupEnv(name); found {
				return value
			}
			if defaultValue != "" {
				return defaultValue
			}
			missing = append(missing, name)
			return ""
		})
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing env variables: %v", missing)
	}
	return nil
}

// Split the string at the first occurrence of sep, and return the part before the separator,
// and the part after the separator.
//
// If the separator is not found, return the string itself, followed by an empty string.
func partition(s, sep string) (string, string) {
	if strings.Contains(s, sep) {
		parts := strings.SplitN(s, sep, 2)
		return parts[0], parts[1]
	}
	return s, ""
}
