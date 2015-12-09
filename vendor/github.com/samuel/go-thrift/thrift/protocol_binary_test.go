// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package thrift

import (
	"bytes"
	"testing"
)

func testProtocol(t *testing.T, r ProtocolReader, w ProtocolWriter) {
	// b := &bytes.Buffer{}
	// r := pr.NewProtocolReader(b)
	// w := pr.NewProtocolWriter(b)

	if err := w.WriteBool(true); err != nil {
		t.Fatalf("write bool true failed: %+v", err)
	}
	if b, err := r.ReadBool(); err != nil {
		t.Fatalf("read bool true failed: %+v", err)
	} else if !b {
		t.Fatal("read bool true returned false")
	}

	if err := w.WriteBool(false); err != nil {
		t.Fatalf("write bool false failed: %+v", err)
	}
	if b, err := r.ReadBool(); err != nil {
		t.Fatalf("read bool false failed: %+v", err)
	} else if b {
		t.Fatal("read bool false returned true")
	}

	if err := w.WriteI16(1234); err != nil {
		t.Fatalf("write i16 failed: %+v", err)
	}
	if v, err := r.ReadI16(); err != nil {
		t.Fatalf("read i16 failed: %+v", err)
	} else if v != 1234 {
		t.Fatalf("read i16 returned %d expected 1234", v)
	}

	if err := w.WriteI32(-1234); err != nil {
		t.Fatalf("write i32 failed: %+v", err)
	}
	if v, err := r.ReadI32(); err != nil {
		t.Fatalf("read i32 failed: %+v", err)
	} else if v != -1234 {
		t.Fatalf("read i32 returned %d expected -1234", v)
	}

	if err := w.WriteI64(-1234); err != nil {
		t.Fatalf("write i64 failed: %+v", err)
	}
	if v, err := r.ReadI64(); err != nil {
		t.Fatalf("read i64 failed: %+v", err)
	} else if v != -1234 {
		t.Fatalf("read i64 returned %d expected -1234", v)
	}

	if err := w.WriteDouble(-0.1234); err != nil {
		t.Fatalf("write double failed: %+v", err)
	}
	if v, err := r.ReadDouble(); err != nil {
		t.Fatalf("read double failed: %+v", err)
	} else if v != -0.1234 {
		t.Fatalf("read double returned %.4f expected -0.1234", v)
	}

	testString := "012345"
	for i := 0; i < 2; i++ {
		if err := w.WriteString(testString); err != nil {
			t.Fatalf("write string failed: %+v", err)
		}
		if v, err := r.ReadString(); err != nil {
			t.Fatalf("read string failed: %+v", err)
		} else if v != testString {
			t.Fatalf("read string returned %s expected '%s'", v, testString)
		}
		testString += "012345"
	}

	// Write a message

	if err := w.WriteMessageBegin("msgName", 2, 123); err != nil {
		t.Fatalf("WriteMessageBegin failed: %+v", err)
	}
	if err := w.WriteStructBegin("struct"); err != nil {
		t.Fatalf("WriteStructBegin failed: %+v", err)
	}

	if err := w.WriteFieldBegin("boolTrue", TypeBool, 1); err != nil {
		t.Fatalf("WriteFieldBegin failed: %+v", err)
	}
	if err := w.WriteBool(true); err != nil {
		t.Fatalf("WriteBool(true) failed: %+v", err)
	}
	if err := w.WriteFieldEnd(); err != nil {
		t.Fatalf("WriteFieldEnd failed: %+v", err)
	}

	if err := w.WriteFieldBegin("boolFalse", TypeBool, 3); err != nil {
		t.Fatalf("WriteFieldBegin failed: %+v", err)
	}
	if err := w.WriteBool(false); err != nil {
		t.Fatalf("WriteBool(false) failed: %+v", err)
	}
	if err := w.WriteFieldEnd(); err != nil {
		t.Fatalf("WriteFieldEnd failed: %+v", err)
	}

	if err := w.WriteFieldBegin("str", TypeString, 2); err != nil {
		t.Fatalf("WriteFieldBegin failed: %+v", err)
	}
	if err := w.WriteString("foo"); err != nil {
		t.Fatalf("WriteString failed: %+v", err)
	}
	if err := w.WriteFieldEnd(); err != nil {
		t.Fatalf("WriteFieldEnd failed: %+v", err)
	}

	if err := w.WriteFieldStop(); err != nil {
		t.Fatalf("WriteStructEnd failed: %+v", err)
	}
	if err := w.WriteStructEnd(); err != nil {
		t.Fatalf("WriteStructEnd failed: %+v", err)
	}
	if err := w.WriteMessageEnd(); err != nil {
		t.Fatalf("WriteMessageEnd failed: %+v", err)
	}

	// Read the message

	if name, mtype, seqID, err := r.ReadMessageBegin(); err != nil {
		t.Fatalf("ReadMessageBegin failed: %+v", err)
	} else if name != "msgName" {
		t.Fatalf("ReadMessageBegin name mismatch: %s != %s", name, "msgName")
	} else if mtype != 2 {
		t.Fatalf("ReadMessageBegin type mismatch: %d != %d", mtype, 2)
	} else if seqID != 123 {
		t.Fatalf("ReadMessageBegin seqID mismatch: %d != %d", seqID, 123)
	}
	if err := r.ReadStructBegin(); err != nil {
		t.Fatalf("ReadStructBegin failed: %+v", err)
	}

	if fieldType, id, err := r.ReadFieldBegin(); err != nil {
		t.Fatalf("ReadFieldBegin failed: %+v", err)
	} else if fieldType != TypeBool {
		t.Fatalf("ReadFieldBegin type mismatch: %d != %d", fieldType, TypeBool)
	} else if id != 1 {
		t.Fatalf("ReadFieldBegin id mismatch: %d != %d", id, 1)
	}
	if v, err := r.ReadBool(); err != nil {
		t.Fatalf("ReaBool failed: %+v", err)
	} else if !v {
		t.Fatalf("ReadBool value mistmatch %+v != %+v", v, true)
	}
	if err := r.ReadFieldEnd(); err != nil {
		t.Fatalf("ReadFieldEnd failed: %+v", err)
	}

	if fieldType, id, err := r.ReadFieldBegin(); err != nil {
		t.Fatalf("ReadFieldBegin failed: %+v", err)
	} else if fieldType != TypeBool {
		t.Fatalf("ReadFieldBegin type mismatch: %d != %d", fieldType, TypeBool)
	} else if id != 3 {
		t.Fatalf("ReadFieldBegin id mismatch: %d != %d", id, 3)
	}
	if v, err := r.ReadBool(); err != nil {
		t.Fatalf("ReaBool failed: %+v", err)
	} else if v {
		t.Fatalf("ReadBool value mistmatch %+v != %+v", v, false)
	}
	if err := r.ReadFieldEnd(); err != nil {
		t.Fatalf("ReadFieldEnd failed: %+v", err)
	}

	if fieldType, id, err := r.ReadFieldBegin(); err != nil {
		t.Fatalf("ReadFieldBegin failed: %+v", err)
	} else if fieldType != TypeString {
		t.Fatalf("ReadFieldBegin type mismatch: %d != %d", fieldType, TypeString)
	} else if id != 2 {
		t.Fatalf("ReadFieldBegin id mismatch: %d != %d", id, 2)
	}
	if v, err := r.ReadString(); err != nil {
		t.Fatalf("ReadString failed: %+v", err)
	} else if v != "foo" {
		t.Fatalf("ReadString value mistmatch %s != %s", v, "foo")
	}
	if err := r.ReadFieldEnd(); err != nil {
		t.Fatalf("ReadFieldEnd failed: %+v", err)
	}

	if err := r.ReadStructEnd(); err != nil {
		t.Fatalf("ReadStructEnd failed: %+v", err)
	}
	if err := r.ReadMessageEnd(); err != nil {
		t.Fatalf("ReadMessageEnd failed: %+v", err)
	}
}

func TestBinaryProtocolBadStringLength(t *testing.T) {
	b := &bytes.Buffer{}
	w := NewBinaryProtocolWriter(b, true)
	r := NewBinaryProtocolReader(b, false)

	// zero string length
	if err := w.WriteI32(0); err != nil {
		t.Fatal(err)
	}
	if st, err := r.ReadString(); err != nil {
		t.Fatal(err)
	} else if st != "" {
		t.Fatal("BinaryProtocol.ReadString didn't return an empty string given a length of 0")
	}

	// negative string length
	if err := w.WriteI32(-1); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ReadString(); err == nil {
		t.Fatal("BinaryProtocol.ReadString didn't return an error given a negative length")
	}
}

func TestBinaryProtocol(t *testing.T) {
	b := &bytes.Buffer{}
	testProtocol(t, NewBinaryProtocolReader(b, false), NewBinaryProtocolWriter(b, true))
	b.Reset()
	testProtocol(t, NewBinaryProtocolReader(b, false), NewBinaryProtocolWriter(b, false))
	b.Reset()
	testProtocol(t, NewBinaryProtocolReader(b, true), NewBinaryProtocolWriter(b, true))
}

func BenchmarkBinaryProtocolReadByte(b *testing.B) {
	buf := &loopingReader{}
	w := NewBinaryProtocolWriter(buf, true)
	r := NewBinaryProtocolReader(buf, false)
	w.WriteByte(123)
	for i := 0; i < b.N; i++ {
		r.ReadByte()
	}
}

func BenchmarkBinaryProtocolReadI32(b *testing.B) {
	buf := &loopingReader{}
	w := NewBinaryProtocolWriter(buf, true)
	r := NewBinaryProtocolReader(buf, false)
	w.WriteI32(1234567890)
	for i := 0; i < b.N; i++ {
		r.ReadI32()
	}
}

func BenchmarkBinaryProtocolWriteByte(b *testing.B) {
	buf := nullWriter(0)
	w := NewBinaryProtocolWriter(buf, true)
	for i := 0; i < b.N; i++ {
		w.WriteByte(1)
	}
}

func BenchmarkBinaryProtocolWriteI32(b *testing.B) {
	buf := nullWriter(0)
	w := NewBinaryProtocolWriter(buf, true)
	for i := 0; i < b.N; i++ {
		w.WriteI32(1)
	}
}

func BenchmarkBinaryProtocolWriteString4(b *testing.B) {
	buf := nullWriter(0)
	w := NewBinaryProtocolWriter(buf, true)
	for i := 0; i < b.N; i++ {
		w.WriteString("test")
	}
}

func BenchmarkBinaryProtocolWriteFullMessage(b *testing.B) {
	buf := nullWriter(0)
	w := NewBinaryProtocolWriter(buf, true)
	for i := 0; i < b.N; i++ {
		w.WriteMessageBegin("", 2, 123)
		w.WriteStructBegin("")
		w.WriteFieldBegin("", TypeBool, 1)
		w.WriteBool(true)
		w.WriteFieldEnd()
		w.WriteFieldBegin("", TypeBool, 3)
		w.WriteBool(false)
		w.WriteFieldEnd()
		w.WriteFieldBegin("", TypeString, 2)
		w.WriteString("foo")
		w.WriteFieldEnd()
		w.WriteFieldStop()
		w.WriteStructEnd()
		w.WriteMessageEnd()
	}
}
