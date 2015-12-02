// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package thrift

import (
	"testing"
)

func TestTagParsing(t *testing.T) {
	id, opts := parseTag("1,foobar,foo")
	if id != 1 {
		t.Fatalf("id = %d, want 1", id)
	}
	for _, tt := range []struct {
		opt  string
		want bool
	}{
		{"foobar", true},
		{"foo", true},
		{"bar", false},
	} {
		if opts.Contains(tt.opt) != tt.want {
			t.Errorf("Contains(%q) = %v", tt.opt, !tt.want)
		}
	}
}
