package treewriter

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/docker/cli/cli/command/formatter/tabwriter"
)

func PrintTree(header []string, rows [][]string) {
	// TODO(thaJeztah): using [][]string doesn't work well for this; we should
	// create a type for this that has "optional" child records that we can
	// recurse over to build the tree.
	tree := make([][]string, 0, len(rows))
	tree = append(tree, header)
	tree = append(tree, rows...)

	buf := bytes.NewBuffer(nil)
	tw := tabwriter.NewWriter(buf, 20, 1, 3, ' ', 0)
	for rowNum, cols := range tree {
		if len(cols) == 0 {
			continue
		}

		// The treePrefix is basically what makes it a "tree" when printing.
		// We shouldn't need to have "columns" though, only know about nesting
		// level, and otherwise have a pre-formatted, tab-terminated string
		// for each row.
		var treePrefix string
		treePrefix, cols = cols[0], cols[1:]
		if treePrefix == "" {
			// Start of new group
			if rowNum > 1 {
				// Print an empty line between groups. We need to write a tab
				// for each column for the tab-writer to give all groups equal
				// widths.
				//
				// FIXME(thaJeztah): if we pass rows as a pre-formatted string,
				//  instead of a []string, we need to know the number of columns
				//  (probably counting  number of tabs would do the trick).
				_, _ = fmt.Fprintln(tw, strings.Repeat("\t", len(cols)))
			}
			_, _ = fmt.Fprintln(tw, strings.Join(cols, "\t"))
		} else {
			_, _ = fmt.Fprintln(tw, treePrefix, strings.Join(cols, "\t"))
		}
	}
	_ = tw.Flush()
	fmt.Println(buf.String())
}
