// Copyright 2016 cli-docs-tool authors
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
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// GenManTree generates a man page for the command and all descendants.
// If SOURCE_DATE_EPOCH is set, in order to allow reproducible package
// builds, we explicitly set the build time to SOURCE_DATE_EPOCH.
func (c *Client) GenManTree(cmd *cobra.Command) error {
	if err := c.loadLongDescription(cmd, "man"); err != nil {
		return err
	}

	if epoch := os.Getenv("SOURCE_DATE_EPOCH"); c.manHeader != nil && epoch != "" {
		unixEpoch, err := strconv.ParseInt(epoch, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid SOURCE_DATE_EPOCH: %v", err)
		}
		now := time.Unix(unixEpoch, 0)
		c.manHeader.Date = &now
	}

	return c.genManTreeCustom(cmd)
}

func (c *Client) genManTreeCustom(cmd *cobra.Command) error {
	for _, sc := range cmd.Commands() {
		if err := c.genManTreeCustom(sc); err != nil {
			return err
		}
	}

	// always disable the addition of [flags] to the usage
	cmd.DisableFlagsInUseLine = true

	// always disable "spf13/cobra" auto gen tag
	cmd.DisableAutoGenTag = true

	// Skip the root command altogether, to prevent generating a useless
	// md file for plugins.
	if c.plugin && !cmd.HasParent() {
		return nil
	}

	// Skip hidden command recursively
	for curr := cmd; curr != nil; curr = curr.Parent() {
		if curr.Hidden {
			log.Printf("INFO: Skipping Man for %q (hidden command)", curr.CommandPath())
			return nil
		}
	}

	log.Printf("INFO: Generating Man for %q", cmd.CommandPath())

	return doc.GenManTreeFromOpts(cmd, doc.GenManTreeOptions{
		Header:           c.manHeader,
		Path:             c.target,
		CommandSeparator: "-",
	})
}
