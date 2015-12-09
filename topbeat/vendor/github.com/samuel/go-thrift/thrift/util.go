// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package thrift

import (
	"strings"
	"unicode"
)

// CamelCase returns the string converted to camel case (e.g. some_name to SomeName)
func CamelCase(s string) string {
	prev := '_'
	return strings.Map(
		func(r rune) rune {
			if r == '_' {
				prev = r
				return -1
			}
			if prev == '_' {
				prev = r
				return unicode.ToUpper(r)
			}
			prev = r
			return r
		}, s)
}
