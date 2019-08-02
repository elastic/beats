// Copyright 2017 Tamás Gulácsi
//
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package goracle

import (
	"encoding/json"
	"testing"

	"github.com/pkg/errors"
)

func TestFromErrorInfo(t *testing.T) {
	errInfo := newErrorInfo(0, "ORA-24315: érvénytelen attribútumtípus\n")
	t.Log("errInfo", errInfo)
	oe := fromErrorInfo(errInfo)
	t.Log("OraErr", oe)
	if oe.Code() != 24315 {
		t.Errorf("got %d, wanted 24315", oe.Code())
	}
}

func TestMarshalJSON(t *testing.T) {
	n := Number("12345.6789")
	b, err := (&n).MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	n = Number("")
	if err = n.UnmarshalJSON(b); err != nil {
		t.Fatal(err)
	}
	t.Log(n.String())

	n = Number("")
	b, err = json.Marshal(struct {
		N Number
		A int
	}{N: n, A: 12})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(b))

	type myStruct struct {
		N interface{}
		A int
	}
	n = Number("")
	ttt := myStruct{N: &n, A: 12}
	b, err = json.Marshal(ttt)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(b))
}

func TestParseTZ(t *testing.T) {
	for k, v := range map[string]int{
		"00:00": 0, "+00:00": 0, "-00:00": 0,
		"01:00": 3600, "+01:00": 3600, "-01:01": -3601,
		"+02:03": 7203,
	} {
		i, err := parseTZ(k)
		if err != nil {
			t.Fatal(errors.Wrap(err, k))
		}
		if i != v {
			t.Errorf("%s. got %d, wanted %d.", k, i, v)
		}
	}
}
