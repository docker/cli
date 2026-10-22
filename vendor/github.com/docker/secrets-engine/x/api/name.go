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
)

var ErrEmptyName = errors.New("empty name")

type Name interface {
	String() string
}

type name struct {
	value string
}

func (n *name) String() string {
	return n.value
}

// NewName creates a new [Name] from a string
// If a validation error occurs, it returns nil and the error.
func NewName(s string) (Name, error) {
	if err := validName(s); err != nil {
		return nil, fmt.Errorf("parsing name: %w", err)
	}
	return &name{
		value: s,
	}, nil
}

// MustNewName creates a new Name as [NewName] does, but panics when a validation
// error occurs.
func MustNewName(s string) Name {
	v, err := NewName(s)
	if err != nil {
		panic(err)
	}
	return v
}

func validName(s string) error {
	if s == "" {
		return ErrEmptyName
	}
	return nil
}
