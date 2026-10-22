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

package api

import (
	"errors"
	"fmt"

	"golang.org/x/mod/semver"
)

var (
	ErrEmptyVersion = errors.New("empty version")
	ErrVPrefix      = errors.New("missing version prefix")
)

type Version interface {
	String() string
}

type version struct {
	value string
}

func (v *version) String() string {
	return v.value
}

// NewVersion creates a new [Version] from a string
// If a validation error occurs, it returns nil and the error.
func NewVersion(s string) (Version, error) {
	if err := valid(s); err != nil {
		return nil, fmt.Errorf("parsing version: %w", err)
	}
	return &version{
		value: s,
	}, nil
}

// MustNewVersion creates a new Version as [NewVersion] does, but panics when a validation
// error occurs.
func MustNewVersion(s string) Version {
	v, err := NewVersion(s)
	if err != nil {
		panic(err)
	}
	return v
}

// Compare returns an integer comparing two versions according to semantic versioning.
// The result will be 0 if a == b, -1 if a < b, or +1 if a > b.
// An invalid semver string is considered less than a valid one.
func Compare(a, b Version) int {
	return semver.Compare(a.String(), b.String())
}

func valid(s string) error {
	if len(s) > 0 && s[0] != 'v' {
		return ErrVPrefix
	}
	if s == "" {
		return ErrEmptyVersion
	}
	if !semver.IsValid(s) {
		return fmt.Errorf("invalid version: %s", s)
	}
	return nil
}
