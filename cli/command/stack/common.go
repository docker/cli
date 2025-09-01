package stack

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/docker/cli/cli/compose/convert"
	"github.com/docker/cli/opts"
	"github.com/moby/moby/api/types/filters"
)

// validateStackName checks if the provided string is a valid stack name (namespace).
// It currently only does a rudimentary check if the string is empty, or consists
// of only whitespace and quoting characters.
func validateStackName(namespace string) error {
	v := strings.TrimFunc(namespace, quotesOrWhitespace)
	if v == "" {
		return fmt.Errorf("invalid stack name: %q", namespace)
	}
	return nil
}

func validateStackNames(namespaces []string) error {
	for _, ns := range namespaces {
		if err := validateStackName(ns); err != nil {
			return err
		}
	}
	return nil
}

func quotesOrWhitespace(r rune) bool {
	return unicode.IsSpace(r) || r == '"' || r == '\''
}

func getStackFilterFromOpt(namespace string, opt opts.FilterOpt) filters.Args {
	filter := opt.Value()
	filter.Add("label", convert.LabelNamespace+"="+namespace)
	return filter
}

func getAllStacksFilter() filters.Args {
	filter := filters.NewArgs()
	filter.Add("label", convert.LabelNamespace)
	return filter
}
