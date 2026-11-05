// Copyright 2026 Docker, Inc.
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

import "strings"

func canonicalKey(toks []token) string {
	var sb strings.Builder
	for i, t := range toks {
		if i > 0 {
			sb.WriteByte('/')
		}
		switch t.kind {
		case tokenStar:
			sb.WriteByte('*')
		case tokenGap:
			sb.WriteString("**")
		case tokenLit:
			sb.WriteString(t.lit)
		}
	}
	return sb.String()
}

// Minimize removes each pattern that another contains (see
// [Pattern.Contains]); when patterns match the same IDs, only the first
// stays. The result preserves input order.
//
// Examples:
//
//	[** a/*]        ->  [**]
//	[**/* */** **]  ->  [**/*]
//	[a/** **/a]     ->  [a/** **/a]
//
// Complexity: O(n²·L²) for n patterns of at most L components each.
func Minimize(patterns []Pattern) []Pattern {
	entries := make([]entry, 0, len(patterns))
	seen := make(map[string]struct{}, len(patterns))
	for _, p := range patterns {
		toks := canonicalize(p.String())
		key := canonicalKey(toks)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		entries = append(entries, newEntry(p, toks))
	}

	result := make([]Pattern, 0, len(entries))
	for i := range entries {
		redundant := false
		for j := range entries {
			if i == j || !subsumes(&entries[j], &entries[i]) {
				continue
			}
			if j < i || !covers(entries[i].toks, entries[j].toks) {
				redundant = true
				break
			}
		}
		if !redundant {
			result = append(result, entries[i].original)
		}
	}
	return result
}

type entry struct {
	original Pattern
	toks     []token
	minLen   int
	hasWild  bool
	hasGaps  bool
	firstLit string
	lastLit  string
}

func newEntry(p Pattern, toks []token) entry {
	e := entry{original: p, toks: toks}
	for _, t := range toks {
		if t.kind != tokenLit {
			e.hasWild = true
		}
		if t.kind == tokenGap {
			e.hasGaps = true
		} else {
			e.minLen++
		}
	}
	if len(toks) > 0 {
		e.firstLit = toks[0].lit
		e.lastLit = toks[len(toks)-1].lit
	}
	return e
}

// subsumes reports whether other matches every ID e matches. It compares
// the precomputed fields first, so most pairs skip the O(L²) covers call.
func subsumes(other, e *entry) bool {
	if !other.hasWild {
		return false
	}
	if other.minLen > e.minLen {
		return false
	}
	if !other.hasGaps && (e.hasGaps || other.minLen != e.minLen) {
		return false
	}
	if other.firstLit != "" && other.firstLit != e.firstLit {
		return false
	}
	if other.lastLit != "" && other.lastLit != e.lastLit {
		return false
	}
	return covers(other.toks, e.toks)
}
