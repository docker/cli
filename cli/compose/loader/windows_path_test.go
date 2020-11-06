package loader

// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// https://github.com/golang/go/blob/master/LICENSE

// The code in this file was copied from the Golang filepath package with some
// small modifications to run it on non-Windows platforms.
// https://github.com/golang/go/blob/1d0e94b1e13d5e8a323a63cd1cc1ef95290c9c36/src/path/filepath/path_test.go#L711-L763

import "testing"

type IsAbsTest struct {
	path  string
	isAbs bool
}

var isabstests = []IsAbsTest{
	{"", false},
	{"/", true},
	{"/usr/bin/gcc", true},
	{"..", false},
	{"/a/../bb", true},
	{".", false},
	{"./", false},
	{"lala", false},
}

var winisabstests = []IsAbsTest{
	{`C:\`, true},
	{`c\`, false},
	{`c::`, false},
	{`c:`, false},
	{`/`, false},
	{`\`, false},
	{`\Windows`, false},
	{`c:a\b`, false},
	{`c:\a\b`, true},
	{`c:/a/b`, true},
	{`\\host\share\foo`, true},
	{`//host/share/foo/bar`, true},
}

func TestIsAbs(t *testing.T) {
	tests := winisabstests

	// All non-windows tests should fail, because they have no volume letter.
	for _, test := range isabstests {
		tests = append(tests, IsAbsTest{test.path, false})
	}
	// All non-windows test should work as intended if prefixed with volume letter.
	for _, test := range isabstests {
		tests = append(tests, IsAbsTest{"c:" + test.path, test.isAbs})
	}

	for _, test := range tests {
		if r := isAbs(test.path); r != test.isAbs {
			t.Errorf("IsAbs(%q) = %v, want %v", test.path, r, test.isAbs)
		}
	}
}
