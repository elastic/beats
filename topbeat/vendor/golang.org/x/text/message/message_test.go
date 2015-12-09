// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package message

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"golang.org/x/text/internal/format"
	"golang.org/x/text/language"
)

type formatFunc func(s fmt.State, v rune)

func (f formatFunc) Format(s fmt.State, v rune) { f(s, v) }

func TestBinding(t *testing.T) {
	testCases := []struct {
		tag   string
		value interface{}
		want  string
	}{
		{"en", 1, "1"},
		{"en", "2", "2"},
		{ // Language is passed.
			"en",
			formatFunc(func(fs fmt.State, v rune) {
				s := fs.(format.State)
				io.WriteString(s, s.Language().String())
			}),
			"en",
		},
	}
	for i, tc := range testCases {
		p := NewPrinter(language.MustParse(tc.tag))
		if got := p.Sprint(tc.value); got != tc.want {
			t.Errorf("%d:%s:Sprint(%v) = %q; want %q", i, tc.tag, tc.value, got, tc.want)
		}
		var buf bytes.Buffer
		p.Fprint(&buf, tc.value)
		if got := buf.String(); got != tc.want {
			t.Errorf("%d:%s:Fprint(%v) = %q; want %q", i, tc.tag, tc.value, got, tc.want)
		}
	}
}
