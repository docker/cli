// Copyright 2025-2026 Docker, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package secrets

import (
	"fmt"
	"strings"
)

type ErrInvalidID struct {
	ID string
}

func (e ErrInvalidID) Error() string {
	return fmt.Sprintf("invalid identifier: %q must match ^[A-Za-z0-9._:-]+(?:/[A-Za-z0-9._:-]+)*$", e.ID)
}

// validIdentifier checks if an identifier is valid without using regexp or unicode.
// Rules:
// - Components separated by '/'
// - Each component is non-empty
// - Only characters A-Z, a-z, 0-9, '.', '_', '-' or ':'
// - No leading, trailing, or double slashes
func validIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}

	componentLen := 0

	for _, r := range s {
		switch {
		case r == '/':
			if componentLen == 0 {
				// Empty component (leading, trailing, or double slash)
				return false
			}
			componentLen = 0
		case isValidRune(r):
			componentLen++
		default:
			return false
		}
	}

	// Final component must not be empty
	return componentLen > 0
}

func isValidRune(c rune) bool {
	return (c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') ||
		c == '.' || c == '-' || c == '_' || c == ':'
}

func split(s string) []string {
	s = strings.Trim(s, "/")
	if s == "" {
		return []string{}
	}
	return strings.Split(s, "/")
}

func match(pattern, path []string) bool {
	pi, si := 0, 0
	for pi < len(pattern) && si < len(path) {
		switch pattern[pi] {
		case "**":
			// Try to consume any number of path segments
			if pi+1 == len(pattern) {
				return true // trailing ** matches rest
			}
			// Try to find a match for the rest of the pattern
			for skip := 0; si+skip <= len(path); skip++ {
				if match(pattern[pi+1:], path[si+skip:]) {
					return true
				}
			}
			return false
		case "*":
			// Match exactly one path component
			pi++
			si++
		default:
			if pattern[pi] != path[si] {
				return false
			}
			pi++
			si++
		}
	}

	// Handle trailing pattern parts (like **)
	for pi < len(pattern) && pattern[pi] == "**" {
		pi++
	}

	return pi == len(pattern) && si == len(path)
}

func valid(id string) error {
	if !validIdentifier(id) {
		return ErrInvalidID{
			ID: id,
		}
	}

	return nil
}

// ID contains a secret identifier.
// Valid secret identifiers must match the format ^[A-Za-z0-9._:-]+(?:/[A-Za-z0-9._:-]+)*$.
//
// For storage, we don't really differentiate much about the ID format but
// by convention we do simple, slash-separated management, providing a
// groupable access control system for management across plugins.
type ID interface {
	// String formats the [ID] as a string
	String() string
	// Match the [ID] against a [Pattern]
	// It checks if a given identifier matches the pattern.
	// - "*" matches a single component
	// - "**" matches zero or more components
	// - "/" is the separator
	Match(pattern Pattern) bool
}

type id string

func (i id) Match(pattern Pattern) bool {
	pathParts := split(string(i))
	patternParts := split(pattern.String())

	return match(patternParts, pathParts)
}

func (i id) String() string {
	return string(i)
}

// ParseID creates a new [ID] from a string
// If a validation error occurs, it returns nil and the error.
// Rules:
// - Components separated by '/'
// - Each component is non-empty
// - Only characters A-Z, a-z, 0-9, '.', '_', '-' or ':'
// - No leading, trailing, or double slashes
func ParseID(s string) (ID, error) {
	if err := valid(s); err != nil {
		return nil, err
	}

	return id(s), nil
}

// MustParseID parses a string into a [ID] and behaves similar to
// [ParseID], however, it panics when the id is invalid
func MustParseID(s string) ID {
	if err := valid(s); err != nil {
		panic(err)
	}
	return id(s)
}
