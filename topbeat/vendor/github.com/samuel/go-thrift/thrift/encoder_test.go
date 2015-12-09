// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package thrift

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
	"time"
)

type TestStruct2 struct {
	Str    string `thrift:"1"`
	Binary []byte `thrift:"2"`
}

func (t *TestStruct2) String() string {
	return fmt.Sprintf("{Str:%s Binary:%+v}", t.Str, t.Binary)
}

type IntSet []int32

func (s *IntSet) EncodeThrift(w ProtocolWriter) error {
	if err := w.WriteByte(byte(len(*s))); err != nil {
		return err
	}
	for _, v := range *s {
		if err := w.WriteI32(v); err != nil {
			return err
		}
	}
	return nil
}

func (s *IntSet) DecodeThrift(r ProtocolReader) error {
	l, err := r.ReadByte()
	if err != nil {
		return err
	}
	sl := (*s)[:0]
	for i := byte(0); i < l; i++ {
		v, err := r.ReadI32()
		if err != nil {
			return err
		}
		sl = append(sl, v*10)
	}
	*s = sl
	return nil
}

type TestStruct struct {
	String   string              `thrift:"1"`
	Int      *int                `thrift:"2"`
	List     []string            `thrift:"3"`
	Map      map[string]string   `thrift:"4"`
	Struct   *TestStruct2        `thrift:"5"`
	List2    []*string           `thrift:"6"`
	Struct2  TestStruct2         `thrift:"7"`
	Binary   []byte              `thrift:"8"`
	Set      []string            `thrift:"9,set"`
	Set2     map[string]struct{} `thrift:"10"`
	Set3     map[string]bool     `thrift:"11,set"`
	Uint32   uint32              `thrift:"12"`
	Uint64   uint64              `thrift:"13"`
	Duration time.Duration       `thrift:"14"`
}

type TestStructRequiredOptional struct {
	RequiredPtr *string `thrift:"1,required"`
	Required    string  `thrift:"2,required"`
	OptionalPtr *string `thrift:"3"`
	Optional    string  `thrift:"4"`
}

type TestEmptyStruct struct{}

type testCustomStruct struct {
	Custom *IntSet `thrift:"1"`
}

func TestKeepEmpty(t *testing.T) {
	buf := &bytes.Buffer{}

	s := struct {
		Str1 string `thrift:"1"`
	}{}
	err := EncodeStruct(NewBinaryProtocolWriter(buf, true), s)
	if err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 1 || buf.Bytes()[0] != 0 {
		t.Fatal("missing keepempty should mean empty fields are not serialized")
	}

	buf.Reset()
	s2 := struct {
		Str1 string `thrift:"1,keepempty"`
	}{}
	err = EncodeStruct(NewBinaryProtocolWriter(buf, true), s2)
	if err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 8 {
		t.Fatal("keepempty should cause empty fields to be serialized")
	}
}

func TestEncodeRequired(t *testing.T) {
	buf := &bytes.Buffer{}

	s := struct {
		Str1 string `thrift:"1,required"`
	}{}
	err := EncodeStruct(NewBinaryProtocolWriter(buf, true), s)
	if err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 8 {
		t.Fatal("Non-pointer required fields that aren't 'keepempty' should be serialized empty")
	}

	buf.Reset()
	s2 := struct {
		Str1 *string `thrift:"1,required"`
	}{}
	err = EncodeStruct(NewBinaryProtocolWriter(buf, true), s2)
	_, ok := err.(*MissingRequiredField)
	if !ok {
		t.Fatalf("Missing required field should throw MissingRequiredField instead of %+v", err)
	}
}

func TestBasics(t *testing.T) {
	i := 123
	str := "bar"
	ts2 := TestStruct2{"qwerty", []byte{1, 2, 3}}
	s := &TestStruct{
		"test",
		&i,
		[]string{"a", "b"},
		map[string]string{"a": "b", "1": "2"},
		&ts2,
		[]*string{&str},
		ts2,
		[]byte{1, 2, 3},
		[]string{"a", "b"},
		map[string]struct{}{"i": struct{}{}, "o": struct{}{}},
		map[string]bool{"q": true, "p": false},
		1<<31 + 2,
		1<<63 + 2,
		time.Second,
	}
	buf := &bytes.Buffer{}

	err := EncodeStruct(NewBinaryProtocolWriter(buf, true), s)
	if err != nil {
		t.Fatal(err)
	}

	s2 := &TestStruct{}
	err = DecodeStruct(NewBinaryProtocolReader(buf, false), s2)
	if err != nil {
		t.Fatal(err)
	}

	// Make sure map[string]bool regards a false value as not-belonging to the set
	delete(s.Set3, "p")

	if !reflect.DeepEqual(s, s2) {
		t.Fatalf("encdec doesn't match: %+v != %+v", s, s2)
	}
}

func TestEncodeRequiredFields(t *testing.T) {
	buf := &bytes.Buffer{}

	// encode nil pointer required field

	s := &TestStructRequiredOptional{nil, "", nil, ""}
	err := EncodeStruct(NewBinaryProtocolWriter(buf, true), s)
	if err == nil {
		t.Fatal("Expected MissingRequiredField exception")
	}
	e, ok := err.(*MissingRequiredField)
	if !ok {
		t.Fatalf("Expected MissingRequiredField exception instead %+v", err)
	}
	if e.StructName != "TestStructRequiredOptional" || e.FieldName != "RequiredPtr" {
		t.Fatalf("Expected MissingRequiredField{'TestStructRequiredOptional', 'RequiredPtr'} instead %+v", e)
	}

	// encode empty non-pointer required field

	str := "foo"
	s = &TestStructRequiredOptional{&str, "", nil, ""}
	err = EncodeStruct(NewBinaryProtocolWriter(buf, true), s)
	if err != nil {
		t.Fatal("Empty non-pointer required fields shouldn't return an error")
	}
}

func TestDecodeRequiredFields(t *testing.T) {
	buf := &bytes.Buffer{}

	s := &TestEmptyStruct{}
	err := EncodeStruct(NewBinaryProtocolWriter(buf, true), s)
	if err != nil {
		t.Fatal("Failed to encode empty struct")
	}

	s2 := &TestStructRequiredOptional{}
	err = DecodeStruct(NewBinaryProtocolReader(buf, false), s2)
	if err == nil {
		t.Fatal("Expected MissingRequiredField exception")
	}
	e, ok := err.(*MissingRequiredField)
	if !ok {
		t.Fatalf("Expected MissingRequiredField exception instead %+v", err)
	}
	if e.StructName != "TestStructRequiredOptional" || e.FieldName != "RequiredPtr" {
		t.Fatalf("Expected MissingRequiredField{'TestStructRequiredOptional', 'RequiredPtr'} instead %+v", e)
	}
}

func TestDecodeUnknownFields(t *testing.T) {
	buf := &bytes.Buffer{}

	str := "foo"
	s := &TestStructRequiredOptional{&str, str, &str, str}
	err := EncodeStruct(NewBinaryProtocolWriter(buf, true), s)
	if err != nil {
		t.Fatal("Failed to encode TestStructRequiredOptional struct")
	}

	s2 := &TestEmptyStruct{}
	err = DecodeStruct(NewBinaryProtocolReader(buf, false), s2)
	if err != nil {
		t.Fatalf("Unknown fields during decode weren't ignored: %+v", err)
	}
}

func TestDecodeCustom(t *testing.T) {
	is := IntSet([]int32{1, 2, 3})
	st := &testCustomStruct{
		Custom: &is,
	}

	buf := &bytes.Buffer{}
	err := EncodeStruct(NewBinaryProtocolWriter(buf, true), st)
	if err != nil {
		t.Fatal("Failed to encode custom struct")
	}

	st2 := &testCustomStruct{}
	err = DecodeStruct(NewBinaryProtocolReader(buf, false), st2)
	if err != nil {
		t.Fatalf("Custom fields during decode failed: %+v", err)
	}
	expected := IntSet([]int32{10, 20, 30})
	if !reflect.DeepEqual(expected, *st2.Custom) {
		t.Fatalf("Custom decode failed expected %+v instead %+v", expected, *st2.Custom)
	}
}

// Benchmarks

func BenchmarkEncodeEmptyStruct(b *testing.B) {
	buf := nullWriter(0)
	st := &struct{}{}
	for i := 0; i < b.N; i++ {
		EncodeStruct(NewBinaryProtocolWriter(buf, true), st)
	}
}

func BenchmarkDecodeEmptyStruct(b *testing.B) {
	b.StopTimer()
	buf1 := &bytes.Buffer{}
	st := &struct{}{}
	EncodeStruct(NewBinaryProtocolWriter(buf1, true), st)
	buf := bytes.NewBuffer(bytes.Repeat(buf1.Bytes(), b.N))
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		DecodeStruct(NewBinaryProtocolReader(buf, false), st)
	}
}

func BenchmarkEncodeSimpleStruct(b *testing.B) {
	buf := nullWriter(0)
	st := &struct {
		Str string `thrift:"1,required"`
		Int int32  `thrift:"2,required"`
	}{
		Str: "test",
		Int: 123,
	}
	for i := 0; i < b.N; i++ {
		EncodeStruct(NewBinaryProtocolWriter(buf, true), st)
	}
}

func BenchmarkDecodeSimpleStruct(b *testing.B) {
	b.StopTimer()
	buf1 := &bytes.Buffer{}
	st := &struct {
		Str string `thrift:"1,required"`
		Int int32  `thrift:"2,required"`
	}{
		Str: "test",
		Int: 123,
	}
	buf := bytes.NewBuffer(bytes.Repeat(buf1.Bytes(), b.N))
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		DecodeStruct(NewBinaryProtocolReader(buf, false), st)
	}
}
