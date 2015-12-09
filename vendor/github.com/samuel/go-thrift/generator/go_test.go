// Copyright 2013 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package main

// import (
// 	"bytes"
// 	"github.com/samuel/go-thrift/parser"
// 	"io"
// 	"regexp"
// 	"testing"
// )

// // Regular expressions
// const (
// 	// http://golang.org/ref/spec#identifier
// 	GO_IDENTIFIER = "[\\pL_][\\pL\\pN_]*"
// )

// // Thrift constants
// const (
// 	THRIFT_SIMPLE = `struct UserProfile {
//   1: i32 uid,
//   2: string name,
//   3: string blurb
// }`
// )

// func GenerateThrift(name string, in io.Reader) (out string, err error) {
// 	var (
// 		p  *parser.Parser
// 		th *parser.Thrift
// 		g  *GoGenerator
// 		b  *bytes.Buffer
// 	)
// 	if th, err = p.Parse(in); err != nil {
// 		return
// 	}
// 	g = &GoGenerator{ThriftFiles: th}
// 	b = new(bytes.Buffer)
// 	if err = g.Generate(name, b); err != nil {
// 		return
// 	}
// 	out = b.String()
// 	return
// }

// func Includes(pattern string, in string) bool {
// 	matched, err := regexp.MatchString(pattern, in)
// 	return matched == true && err == nil
// }

// // Generated package names should be valid identifiers.
// // Per: http://golang.org/ref/spec#Package_clause
// func TestGeneratesValidPackageNames(t *testing.T) {
// 	var (
// 		in     *bytes.Buffer
// 		out    string
// 		err    error
// 		tests  map[string]string
// 		is_err bool
// 	)
// 	in = bytes.NewBufferString(THRIFT_SIMPLE)
// 	tests = map[string]string{
// 		"foo-bar": "foo_bar",
// 		"_foo":    "_foo",
// 		"fooαβ":   "fooαβ",
// 		"0foo":    "_0foo",
// 	}
// 	for test, expected := range tests {
// 		if out, err = GenerateThrift(test, in); err != nil {
// 			t.Fatalf("Could not generate Thrift: %v", err)
// 		}
// 		if !Includes("package "+GO_IDENTIFIER+"\n", out) {
// 			t.Errorf("Couldn't find valid package for test %v", test)
// 			is_err = true
// 		}
// 		if !Includes("package "+expected+"\n", out) {
// 			t.Errorf("Couldn't find expected package '%v' for test %v", expected, test)
// 			is_err = true
// 		}
// 		if is_err {
// 			t.Logf("Problem with generated Thrift:\n%v\n", out)
// 			is_err = false
// 		}
// 	}
// }
