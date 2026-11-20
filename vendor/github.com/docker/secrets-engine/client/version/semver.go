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

package version

import "strings"

type semver struct {
	major, minor, patch string
	prerelease          []string
}

func parseSemver(v string) (semver, bool) {
	s, ok := strings.CutPrefix(v, "v")
	if !ok {
		return semver{}, false
	}
	s, build, hasBuild := strings.Cut(s, "+")
	if hasBuild && !validIdentifiers(build, false) {
		return semver{}, false
	}
	core, pre, hasPre := strings.Cut(s, "-")
	if hasPre && !validIdentifiers(pre, true) {
		return semver{}, false
	}
	parts := strings.Split(core, ".")
	if len(parts) > 3 || (len(parts) < 3 && (hasPre || hasBuild)) {
		return semver{}, false
	}
	for _, p := range parts {
		if !isNumeric(p) || (len(p) > 1 && p[0] == '0') {
			return semver{}, false
		}
	}
	for len(parts) < 3 {
		parts = append(parts, "0")
	}
	sv := semver{major: parts[0], minor: parts[1], patch: parts[2]}
	if hasPre {
		sv.prerelease = strings.Split(pre, ".")
	}
	return sv, true
}

func validIdentifiers(s string, prerelease bool) bool {
	for id := range strings.SplitSeq(s, ".") {
		if id == "" {
			return false
		}
		for i := 0; i < len(id); i++ {
			if !isIdentifierChar(id[i]) {
				return false
			}
		}
		if prerelease && len(id) > 1 && id[0] == '0' && isNumeric(id) {
			return false
		}
	}
	return true
}

func isIdentifierChar(c byte) bool {
	switch {
	case '0' <= c && c <= '9', 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z', c == '-':
		return true
	}
	return false
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func compareNumeric(a, b string) int {
	if c := len(a) - len(b); c != 0 {
		return sign(c)
	}
	return strings.Compare(a, b)
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	}
	return 0
}

func comparePrerelease(a, b []string) int {
	switch {
	case len(a) == 0 && len(b) == 0:
		return 0
	case len(a) == 0:
		return 1
	case len(b) == 0:
		return -1
	}
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] == b[i] {
			continue
		}
		an, bn := isNumeric(a[i]), isNumeric(b[i])
		switch {
		case an && bn:
			return compareNumeric(a[i], b[i])
		case an:
			return -1
		case bn:
			return 1
		}
		return strings.Compare(a[i], b[i])
	}
	return sign(len(a) - len(b))
}

func compareSemver(a, b string) int {
	va, okA := parseSemver(a)
	vb, okB := parseSemver(b)
	switch {
	case !okA && !okB:
		return 0
	case !okA:
		return -1
	case !okB:
		return 1
	}
	if c := compareNumeric(va.major, vb.major); c != 0 {
		return c
	}
	if c := compareNumeric(va.minor, vb.minor); c != 0 {
		return c
	}
	if c := compareNumeric(va.patch, vb.patch); c != 0 {
		return c
	}
	return comparePrerelease(va.prerelease, vb.prerelease)
}
