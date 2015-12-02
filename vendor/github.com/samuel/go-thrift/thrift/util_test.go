// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package thrift

import (
	"bytes"
	"testing"
)

type loopingReader struct {
	bytes  []byte
	offset int
}

func (r *loopingReader) Write(b []byte) (int, error) {
	r.bytes = append(r.bytes, b...)
	return len(b), nil
}

func (r *loopingReader) Read(b []byte) (int, error) {
	n := len(r.bytes)
	end := r.offset + len(b)
	if end > n {
		oldOffset := r.offset
		r.offset = 0
		copy(b, r.bytes[oldOffset:])
		return n - oldOffset, nil
	}
	copy(b, r.bytes[r.offset:end])
	if end == n {
		r.offset = 0
	} else {
		r.offset = end
	}
	return len(b), nil
}

type nullWriter int

func (w nullWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

func TestCamelCase(t *testing.T) {
	cases := map[string]string{
		"test":            "Test",
		"Foo":             "Foo",
		"foo_bar":         "FooBar",
		"FooBar":          "FooBar",
		"test__ing":       "TestIng",
		"three_part_word": "ThreePartWord",
		"FOOBAR":          "FOOBAR",
		"TESTing":         "TESTing",
	}
	for k, v := range cases {
		if CamelCase(k) != v {
			t.Fatalf("%s did not properly CamelCase: %s", k, CamelCase(k))
		}
	}
}

func BenchmarkCamelCase(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CamelCase("foo_bar")
	}
}

func TestLoopingReader(t *testing.T) {
	b := make([]byte, 16)
	r := &loopingReader{[]byte{}, 0}
	if n, _ := r.Read(b); n != 0 {
		t.Fatalf("Empty loopingReader should return 0 bytes on Read instead of %d", n)
	}
	r.Write([]byte{1, 2})
	if n, _ := r.Read(b); n != 2 {
		t.Fatalf("loopingReader should return all bytes")
	} else if bytes.Compare(b[:2], []byte{1, 2}) != 0 {
		t.Fatalf("loopingReader output didn't match for full read")
	}
	b[0] = 0
	b[1] = 0
	if n, _ := r.Read(b[:1]); n != 1 {
		t.Fatalf("loopingReader should return 1 byte for non-full read instead of %d", n)
	} else if bytes.Compare(b[:2], []byte{1, 0}) != 0 {
		t.Fatalf("loopingReader output didn't match for non-full read")
	}
	if n, _ := r.Read(b[:1]); n != 1 {
		t.Fatalf("loopingReader should return 1 byte for non-full read2 instead of %d", n)
	} else if bytes.Compare(b[:2], []byte{2, 0}) != 0 {
		t.Fatalf("loopingReader output didn't match for non-full read2, returned %+v instead pf %+v", b[:2], []byte{2, 0})
	}
	if n, _ := r.Read(b[:1]); n != 1 {
		t.Fatalf("loopingReader should return 1 byte for non-full read3 instead of %d", n)
	} else if bytes.Compare(b[:2], []byte{1, 0}) != 0 {
		t.Fatalf("loopingReader output didn't match for non-full read3")
	}
}
