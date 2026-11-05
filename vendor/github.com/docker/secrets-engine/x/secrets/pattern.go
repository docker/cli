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
	"errors"
	"fmt"
	"strings"
)

var ErrInvalidPattern = errors.New("invalid pattern")

// validPattern reports whether s is a valid pattern: non-empty '/'-separated
// components of A-Z, a-z, 0-9, '.', '-', '_', ':', where a component may
// instead be '*' or '**'.
func validPattern(s string) bool {
	if len(s) == 0 {
		return false
	}

	componentLen := 0
	wildcardLen := 0

	for _, r := range s {
		switch {
		case r == '/':
			if !isValidComponentMatcher(componentLen, wildcardLen) {
				return false
			}
			componentLen = 0
			wildcardLen = 0
		case isValidPatternRune(r):
			componentLen++
			if r == '*' {
				wildcardLen++
			}
		default:
			return false
		}
	}
	// Final component
	return isValidComponentMatcher(componentLen, wildcardLen)
}

func isValidComponentMatcher(componentLen, wildcardLen int) bool {
	if wildcardLen > 2 {
		// No more than two wildcards per component
		return false
	}
	if wildcardLen > 0 && wildcardLen != componentLen {
		// Wildcard can't be mixed with other characters in the same component
		return false
	}
	// Component must not be empty
	return componentLen > 0
}

func isValidPatternRune(c rune) bool {
	return isValidRune(c) || c == '*'
}

// Pattern matches secret IDs. It follows the ID validation rules, except
// that '*' matches one component and '**' matches zero or more.
// Below, matches(p) denotes the set of IDs a pattern p matches.
type Pattern interface {
	// Match reports whether the pattern matches id.
	// Complexity: O(n*m^k), where id has n components and the pattern has
	// m components and k occurrences of '**'.
	Match(id ID) bool
	// Contains reports whether every ID that [other] matches can also be
	// matched by the pattern: the set of IDs [other] matches is contained
	// in the set the pattern matches, i.e., matches(other) ⊆ matches(p).
	//
	// Examples:
	//   - docker/** contains docker/*/mcp/*: '**' is more general than
	//     '*/mcp/*'.
	//   - docker/proj1/** does not contain docker/*/mcp/*: docker/*/mcp/*
	//     matches docker/proj2/mcp/x, but docker/proj1/** does not.
	//
	// Complexity: O(n*m)
	Contains(other Pattern) bool
	// Overlaps reports whether the pattern and [other] match at least one
	// ID in common, i.e., matches(p) ∩ matches(other) ≠ ∅.
	//
	// Examples:
	//   - docker/*/mcp/* and docker/proj1/** overlap: both match e.g. docker/proj1/mcp/x.
	//   - bar/** and foo/** do not overlap: an ID cannot begin with both bar and foo.
	//
	// Complexity: O(n*m)
	Overlaps(other Pattern) bool
	// String returns the pattern text.
	String() string

	ExpandID(other ID) (ID, error)
	ExpandPattern(other Pattern) (Pattern, error)
}

type pattern string

func (p pattern) Match(id ID) bool {
	pathParts := split(id.String())
	patternParts := split(string(p))

	return match(patternParts, pathParts)
}

func (p pattern) Contains(other Pattern) bool {
	return covers(canonicalize(string(p)), canonicalize(other.String()))
}

func (p pattern) Overlaps(other Pattern) bool {
	return compatible(canonicalize(string(p)), canonicalize(other.String()))
}

type tokenKind uint8

const (
	tokenLit tokenKind = iota
	tokenStar
	tokenGap
)

type token struct {
	kind tokenKind
	lit  string
}

// canonicalize tokenizes a pattern so that two patterns match the same set
// of IDs exactly when their canonical tokens are equal. Two rewrites give
// that property:
//
//   - In a run of consecutive wildcard components, the '*' move to the
//     front and the '**' merge into one at the end: order within a run
//     does not affect what it matches, and one '**' already matches any
//     number of components.
//   - A pattern consisting only of '**' becomes "*/**": an ID always has
//     at least one component, so both match the same IDs, and the first
//     rewrite already turns the equivalent spelling "**/*" into "*/**".
//
// Examples:
//
//	a/a/** and a/a/**/**  ->  [a a **]
//	a/**/*/**/b           ->  [a * ** b]
//	** and **/*           ->  [* **]
func canonicalize(s string) []token {
	parts := split(s)
	toks := make([]token, 0, len(parts))
	stars, gap := 0, false
	flush := func() {
		for range stars {
			toks = append(toks, token{kind: tokenStar})
		}
		if gap {
			toks = append(toks, token{kind: tokenGap})
		}
		stars, gap = 0, false
	}
	for _, part := range parts {
		switch part {
		case "*":
			stars++
		case "**":
			gap = true
		default:
			flush()
			toks = append(toks, token{kind: tokenLit, lit: part})
		}
	}
	flush()
	if len(toks) == 1 && toks[0].kind == tokenGap {
		toks = []token{{kind: tokenStar}, {kind: tokenGap}}
	}
	return toks
}

// covers reports whether p matches every ID q matches, for canonical tokens.
// The component alphabet is unbounded, so a literal in p never covers an '*'
// or '**' in q.
func covers(p, q []token) bool {
	np, nq := len(p), len(q)
	prev := make([]bool, nq+1)
	cur := make([]bool, nq+1)
	prev[nq] = true
	for i := np - 1; i >= 0; i-- {
		cur[nq] = p[i].kind == tokenGap && prev[nq]
		for j := nq - 1; j >= 0; j-- {
			switch {
			case p[i].kind == tokenGap:
				cur[j] = prev[j] || cur[j+1]
			case q[j].kind == tokenGap:
				cur[j] = p[i].kind == tokenStar && cur[j+1] && prev[j]
			case p[i].kind == tokenStar:
				cur[j] = prev[j+1]
			default:
				cur[j] = q[j].kind == tokenLit && p[i].lit == q[j].lit && prev[j+1]
			}
		}
		prev, cur = cur, prev
	}
	return prev[0]
}

// compatible reports whether p and q match at least one ID in common, for
// canonical tokens. A common ID aligns both token lists over its
// components: wildcards accept anything, so only differing literals or
// mismatched lengths (* vs */foo) rule one out.
func compatible(p, q []token) bool {
	np, nq := len(p), len(q)
	prev := make([]bool, nq+1)
	cur := make([]bool, nq+1)
	prev[nq] = true
	for j := nq - 1; j >= 0; j-- {
		prev[j] = q[j].kind == tokenGap && prev[j+1]
	}
	for i := np - 1; i >= 0; i-- {
		cur[nq] = p[i].kind == tokenGap && prev[nq]
		for j := nq - 1; j >= 0; j-- {
			switch {
			case p[i].kind == tokenGap || q[j].kind == tokenGap:
				cur[j] = prev[j] || cur[j+1]
			case p[i].kind == tokenLit && q[j].kind == tokenLit && p[i].lit != q[j].lit:
				cur[j] = false
			default:
				cur[j] = prev[j+1]
			}
		}
		prev, cur = cur, prev
	}
	return prev[0]
}

func (p pattern) String() string {
	return string(p)
}

// ParsePattern parses s into a [Pattern]: non-empty '/'-separated components
// of A-Z, a-z, 0-9, '.', '-', '_', ':', where a component may instead be '*'
// or '**'. It returns [ErrInvalidPattern] for anything else.
func ParsePattern(s string) (Pattern, error) {
	if !validPattern(s) {
		return nil, ErrInvalidPattern
	}
	return pattern(s), nil
}

// MustParsePattern is like [ParsePattern] but panics on an invalid pattern.
func MustParsePattern(s string) Pattern {
	if !validPattern(s) {
		panic(ErrInvalidPattern)
	}
	return pattern(s)
}

func (p pattern) ExpandID(other ID) (ID, error) {
	val, err := replace1(string(p), other.String())
	if err != nil {
		return nil, err
	}
	return ParseID(val)
}

func (p pattern) ExpandPattern(other Pattern) (Pattern, error) {
	val, err := replace1(string(p), other.String())
	if err != nil {
		return nil, err
	}
	return pattern(val), err
}

func replace1(original, other string) (string, error) {
	components := split(original)
	var candidates []int
	for idx, val := range components {
		if val == "**" {
			candidates = append(candidates, idx)
		}
	}
	if len(candidates) != 1 {
		return "", fmt.Errorf("expand only supports one expansion glob, pattern %s has %d", original, len(candidates))
	}
	components[candidates[0]] = other
	return strings.Join(components, "/"), nil
}
