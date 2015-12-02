// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package thrift

import (
	"bytes"
	"testing"
)

type ClosingBuffer struct {
	*bytes.Buffer
}

func (c *ClosingBuffer) Close() error {
	return nil
}

func TestFramed(t *testing.T) {
	buf := &ClosingBuffer{&bytes.Buffer{}}

	framed := NewFramedReadWriteCloser(buf, 1024)
	if _, err := framed.Write([]byte{1, 2, 3, 4}); err != nil {
		t.Fatalf("Framed: error on Write %s", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("Framed: wrote %d bytes before flush", buf.Len())
	}
	if err := framed.Flush(); err != nil {
		t.Fatalf("Framed: error on Flush %s", err)
	}
	if buf.Len() != 8 {
		t.Fatalf("Framed: wrote (%d) other than 8 bytes after flush", buf.Len())
	}
	if err := framed.Flush(); err != nil {
		t.Fatalf("Framed: error on Flush %s", err)
	}
	if buf.Len() != 8 {
		t.Fatalf("Framed: flush didn't clear write buffer")
	}

	out := buf.Bytes()
	expected := []byte{0, 0, 0, 4, 1, 2, 3, 4}
	if bytes.Compare(out, expected) != 0 {
		t.Fatalf("Framed: expected output %+v but got %+v", expected, out)
	}

	buf = &ClosingBuffer{bytes.NewBuffer([]byte{0, 0, 0, 2, 5, 6})}
	framed = NewFramedReadWriteCloser(buf, 1024)
	out = make([]byte, 4)
	n, err := framed.Read(out[:4])
	if err != nil {
		t.Fatalf("Framed: error from Read %s", err)
	}
	if n != 2 {
		t.Fatalf("Framed: expected read count of 2 instead %d", n)
	}
	if out[0] != 5 || out[1] != 6 {
		t.Fatalf("Framed: expected {5,6} from Read instead %+v", out[:2])
	}
}
