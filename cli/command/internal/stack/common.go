package stack

import (
	"fmt"
	"strings"
	"unicode"
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
