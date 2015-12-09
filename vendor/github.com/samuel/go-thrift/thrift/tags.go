// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package thrift

import (
	"strconv"
	"strings"
)

// tagOptions is the string following a comma in a struct field's "thrift"
// tag, or the empty string. It does not include the leading comma.
type tagOptions string

// parseTag splits a struct field's thrift tag into its id and
// comma-separated options.
func parseTag(tag string) (int, tagOptions) {
	if idx := strings.Index(tag, ","); idx != -1 {
		id, _ := strconv.Atoi(tag[:idx])
		return id, tagOptions(tag[idx+1:])
	}
	id, _ := strconv.Atoi(tag)
	return id, tagOptions("")
}

// Contains returns whether checks that a comma-separated list of options
// contains a particular substr flag. substr must be surrounded by a
// string boundary or commas.
func (o tagOptions) Contains(optionName string) bool {
	if len(o) == 0 {
		return false
	}
	s := string(o)
	for s != "" {
		var next string
		i := strings.Index(s, ",")
		if i >= 0 {
			s, next = s[:i], s[i+1:]
		}
		if s == optionName {
			return true
		}
		s = next
	}
	return false
}
