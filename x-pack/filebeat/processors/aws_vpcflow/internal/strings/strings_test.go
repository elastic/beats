// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package strings

import (
	"testing"
	"unicode"
)

func eq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

var faces = "☺☻☹"

type FieldsTest struct {
	s string
	a []string
}

var fieldstests = []FieldsTest{
	{"", []string{}},
	{" ", []string{}},
	{" \t ", []string{}},
	{"\u2000", []string{}},
	{"  abc  ", []string{"abc"}},
	{"1 2 3 4", []string{"1", "2", "3", "4"}},
	{"1  2  3  4", []string{"1", "2", "3", "4"}},
	{"1\t\t2\t\t3\t4", []string{"1", "2", "3", "4"}},
	{"1\u20002\u20013\u20024", []string{"1", "2", "3", "4"}},
	{"\u2000\u2001\u2002", []string{}},
	{"\n™\t™\n", []string{"™", "™"}},
	{"\n\u20001™2\u2000 \u2001 ™", []string{"1™2", "™"}},
	{"\n1\uFFFD \uFFFD2\u20003\uFFFD4", []string{"1\uFFFD", "\uFFFD2", "3\uFFFD4"}},
	{"1\xFF\u2000\xFF2\xFF \xFF", []string{"1\xFF", "\xFF2\xFF", "\xFF"}},
	{faces, []string{faces}},
}

func TestFields(t *testing.T) {
	var dst [4]string
	for _, tt := range fieldstests {
		n, err := Fields(dst[:], tt.s)
		if err != nil {
			t.Fatal(err)
		}
		if !eq(dst[:n], tt.a) {
			t.Errorf("Fields(%q) = %v; want %v", tt.s, dst[:n], tt.a)
			continue
		}
		if len(tt.a) != n {
			t.Errorf("Return count n = %d; want %d", n, len(tt.a))
		}
	}

	// Smaller
	var smallDst [2]string
	for _, tt := range fieldstests {
		n, err := Fields(smallDst[:], tt.s)
		if err == errTooManySubstrings { //nolint:errorlint // errTooManySubstrings is never wrapped.
			if len(tt.a) > len(smallDst) {
				continue
			}
		}
		if err != nil {
			t.Fatal(err)
		}

		if !eq(smallDst[:n], tt.a[:n]) {
			t.Errorf("Fields(%q) = %v; want %v", tt.s, smallDst[:n], tt.a)
			continue
		}
	}
}

var FieldsFuncTests = []FieldsTest{
	{"", []string{}},
	{"XX", []string{}},
	{"XXhiXXX", []string{"hi"}},
	{"aXXbXXXcX", []string{"a", "b", "c"}},
}

//nolint:errorlint // errTooManySubstrings is never wrapped.
func TestFieldsFunc(t *testing.T) {
	var dst [4]string
	for _, tt := range fieldstests {
		n, err := fieldsFunc(dst[:], tt.s, unicode.IsSpace)
		if err != nil {
			t.Fatal(err)
		}
		if !eq(dst[:n], tt.a) {
			t.Errorf("FieldsFunc(%q, unicode.IsSpace) = %v; want %v", tt.s, dst, tt.a)
			continue
		}
		if len(tt.a) != n {
			t.Errorf("Return count n = %d; want %d", n, len(tt.a))
		}
	}
	pred := func(c rune) bool { return c == 'X' }
	for _, tt := range FieldsFuncTests {
		n, err := fieldsFunc(dst[:], tt.s, pred)
		if err != nil {
			t.Fatal(err)
		}
		if !eq(dst[:n], tt.a) {
			t.Errorf("FieldsFunc(%q) = %v, want %v", tt.s, dst[:n], tt.a)
		}
		if len(tt.a) != n {
			t.Errorf("Return count n = %d; want %d", n, len(tt.a))
		}
	}

	// Smaller
	var smallDst [2]string
	for _, tt := range fieldstests {
		n, err := Fields(smallDst[:], tt.s)
		if err == errTooManySubstrings {
			if len(tt.a) > len(smallDst) {
				continue
			}
		}
		if err != nil {
			t.Fatal(err)
		}

		if !eq(smallDst[:n], tt.a[:n]) {
			t.Errorf("Fields(%q) = %v; want %v", tt.s, smallDst[:n], tt.a)
			continue
		}
	}
	for _, tt := range FieldsFuncTests {
		n, err := fieldsFunc(smallDst[:], tt.s, pred)
		if err == errTooManySubstrings {
			if len(tt.a) > len(smallDst) {
				continue
			}
		}
		if err != nil {
			t.Fatal(err)
		}

		if !eq(smallDst[:n], tt.a[:n]) {
			t.Errorf("Fields(%q) = %v; want %v", tt.s, smallDst[:n], tt.a)
			continue
		}
	}
}
