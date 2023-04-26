// Copyright 2017 cli-docs-tool authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package clidocstool

import (
	"errors"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// Options defines options for cli-docs-tool
type Options struct {
	Root      *cobra.Command
	SourceDir string
	TargetDir string
	Plugin    bool
}

// Client represents an active cli-docs-tool object
type Client struct {
	root   *cobra.Command
	source string
	target string
	plugin bool
}

// New initializes a new cli-docs-tool client
func New(opts Options) (*Client, error) {
	if opts.Root == nil {
		return nil, errors.New("root cmd required")
	}
	if len(opts.SourceDir) == 0 {
		return nil, errors.New("source dir required")
	}
	c := &Client{
		root:   opts.Root,
		source: opts.SourceDir,
		plugin: opts.Plugin,
	}
	if len(opts.TargetDir) == 0 {
		c.target = c.source
	} else {
		c.target = opts.TargetDir
	}
	if err := os.MkdirAll(c.target, 0755); err != nil {
		return nil, err
	}
	return c, nil
}

// GenAllTree creates all structured ref files for this command and
// all descendants in the directory given.
func (c *Client) GenAllTree() error {
	var err error
	if err = c.GenMarkdownTree(c.root); err != nil {
		return err
	}
	if err = c.GenYamlTree(c.root); err != nil {
		return err
	}
	return nil
}

func fileExists(f string) bool {
	info, err := os.Stat(f)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func copyFile(src string, dst string) error {
	sf, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sf.Close()
	df, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer df.Close()
	_, err = io.Copy(df, sf)
	return err
}

func getAliases(cmd *cobra.Command) []string {
	if a := cmd.Annotations["aliases"]; a != "" {
		aliases := strings.Split(a, ",")
		for i := 0; i < len(aliases); i++ {
			aliases[i] = strings.TrimSpace(aliases[i])
		}
		return aliases
	}
	if len(cmd.Aliases) == 0 {
		return cmd.Aliases
	}

	var parentPath string
	if cmd.HasParent() {
		parentPath = cmd.Parent().CommandPath() + " "
	}
	aliases := []string{cmd.CommandPath()}
	for _, a := range cmd.Aliases {
		aliases = append(aliases, parentPath+a)
	}
	return aliases
}
