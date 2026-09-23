// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.26

// Package hostmatch implements matching of registry hostnames against
// patterns containing "*" wildcards, as used for keys in the "auths" and
// "credHelpers" sections of the CLI configuration file.
package hostmatch

import (
	"iter"
	"strings"
)

// IsPattern reports whether key is a valid wildcard host pattern.
//
// A pattern is a hostname (optionally including ":port") in which one or more
// labels contain a "*" wildcard, for example "*.example.com" or
// "*.dkr.ecr.*.amazonaws.com". A wildcard matches any sequence of characters
// within a single label; it never matches a ".".
//
// To prevent patterns from matching an overly broad set of registries, the
// last two labels (the registrable domain and top-level domain, and port, if
// any) must not contain a wildcard; "*.com" and "example.*" are not valid
// patterns. Keys containing a scheme or path are not valid patterns either.
func IsPattern(key string) bool {
	if !strings.Contains(key, "*") || strings.Contains(key, "/") {
		return false
	}
	labels := strings.Split(key, ".")
	if len(labels) < 3 {
		return false
	}
	for _, label := range labels {
		if label == "" {
			return false
		}
	}
	return !strings.Contains(labels[len(labels)-2], "*") && !strings.Contains(labels[len(labels)-1], "*")
}

// Match reports whether host matches pattern. It returns false if pattern
// is not a valid pattern (see [IsPattern]) or if host is not a plain
// hostname (optionally including ":port").
func Match(pattern, host string) bool {
	if !IsPattern(pattern) || strings.Contains(host, "/") {
		return false
	}
	patternLabels := strings.Split(pattern, ".")
	hostLabels := strings.Split(host, ".")
	if len(patternLabels) != len(hostLabels) {
		return false
	}
	for i, label := range hostLabels {
		if label == "" || !matchLabel(patternLabels[i], label) {
			return false
		}
	}
	return true
}

// Best returns the most specific pattern in keys that matches host. Keys that
// are not valid patterns are ignored. Patterns are ranked by the number of
// non-wildcard characters they contain; ties are broken by lexical order, so
// that the result is deterministic.
func Best(keys iter.Seq[string], host string) (string, bool) {
	var (
		best      string
		bestScore = -1
	)
	for key := range keys {
		if !Match(key, host) {
			continue
		}
		score := len(key) - strings.Count(key, "*")
		if score > bestScore || (score == bestScore && key < best) {
			best, bestScore = key, score
		}
	}
	return best, bestScore >= 0
}

// matchLabel reports whether a single hostname label matches the given
// pattern label, in which "*" matches any (possibly empty) sequence of
// characters.
func matchLabel(pattern, label string) bool {
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return pattern == label
	}
	first, last := parts[0], parts[len(parts)-1]
	if len(label) < len(first)+len(last) || !strings.HasPrefix(label, first) || !strings.HasSuffix(label, last) {
		return false
	}
	label = label[len(first) : len(label)-len(last)]
	for _, part := range parts[1 : len(parts)-1] {
		i := strings.Index(label, part)
		if i < 0 {
			return false
		}
		label = label[i+len(part):]
	}
	return true
}
