// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package thrift

import (
	"reflect"
	"testing"
)

type EncodeFieldsTestStruct struct {
	F1  string              `json:"f1" thrift:"1"`
	F2  string              `thrift:"2" json:"f2"`
	Set map[string]struct{} `thrift:"3"`
}

func TestEncodeFields(t *testing.T) {
	s := EncodeFieldsTestStruct{}
	m := encodeFields(reflect.TypeOf(s))
	if len(m.fields) != 3 {
		t.Fatalf("Did not find all fields. %d fields, expected 3 fields", len(m.fields))
	}
	if m.fields[3].fieldType != TypeSet {
		t.Fatalf("Type map[...]struct{} not handled as a Set")
	}
}
