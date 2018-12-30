// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package mapval

import (
	"reflect"
	"testing"

	"github.com/elastic/beats/libbeat/common"
)

func TestPathComponentType_String(t *testing.T) {
	tests := []struct {
		name string
		pct  pathComponentType
		want string
	}{
		{
			"Should return the correct type",
			pcMapKey,
			"map",
		},
		{
			"Should return the correct type",
			pcSliceIdx,
			"slice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pct.String(); got != tt.want {
				t.Errorf("pathComponentType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPathComponent_String(t *testing.T) {
	type fields struct {
		Type  pathComponentType
		Key   string
		Index int
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			"Map key should return a literal",
			fields{pcMapKey, "foo", 0},
			"foo",
		},
		{
			"Array index should return a bracketed number",
			fields{pcSliceIdx, "", 123},
			"[123]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := pathComponent{
				Type:  tt.fields.Type,
				Key:   tt.fields.Key,
				Index: tt.fields.Index,
			}
			if got := pc.String(); got != tt.want {
				t.Errorf("pathComponent.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPath_ExtendSlice(t *testing.T) {
	type args struct {
		index int
	}
	tests := []struct {
		name string
		p    path
		args args
		want path
	}{
		{
			"Extending an empty slice",
			path{},
			args{123},
			path{pathComponent{pcSliceIdx, "", 123}},
		},
		{
			"Extending a non-empty slice",
			path{pathComponent{pcMapKey, "foo", -1}},
			args{123},
			path{pathComponent{pcMapKey, "foo", -1}, pathComponent{pcSliceIdx, "", 123}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.extendSlice(tt.args.index); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("path.extendSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPath_ExtendMap(t *testing.T) {
	type args struct {
		key string
	}
	tests := []struct {
		name string
		p    path
		args args
		want path
	}{
		{
			"Extending an empty slice",
			path{},
			args{"foo"},
			path{pathComponent{pcMapKey, "foo", -1}},
		},
		{
			"Extending a non-empty slice",
			path{}.extendMap("foo"),
			args{"bar"},
			path{pathComponent{pcMapKey, "foo", -1}, pathComponent{pcMapKey, "bar", -1}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.extendMap(tt.args.key); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("path.extendMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPath_Concat(t *testing.T) {
	tests := []struct {
		name string
		p    path
		arg  path
		want path
	}{
		{
			"simple",
			path{}.extendMap("foo"),
			path{}.extendSlice(123),
			path{}.extendMap("foo").extendSlice(123),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.concat(tt.arg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("path.concat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPath_String(t *testing.T) {
	tests := []struct {
		name string
		p    path
		want string
	}{
		{
			"empty",
			path{},
			"",
		},
		{
			"one element",
			path{}.extendMap("foo"),
			"foo",
		},
		{
			"complex",
			path{}.extendMap("foo").extendSlice(123).extendMap("bar"),
			"foo.[123].bar",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.String(); got != tt.want {
				t.Errorf("path.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPath_Last(t *testing.T) {
	tests := []struct {
		name string
		p    path
		want *pathComponent
	}{
		{
			"empty path",
			path{},
			nil,
		},
		{
			"one element",
			path{}.extendMap("foo"),
			&pathComponent{pcMapKey, "foo", -1},
		},
		{
			"many elements",
			path{}.extendMap("foo").extendMap("bar").extendSlice(123),
			&pathComponent{pcSliceIdx, "", 123},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.last(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("path.last() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPath_GetFrom(t *testing.T) {
	fooPath := path{}.extendMap("foo")
	complexPath := path{}.extendMap("foo").extendSlice(0).extendMap("bar").extendSlice(1)
	tests := []struct {
		name       string
		p          path
		arg        common.MapStr
		wantValue  interface{}
		wantExists bool
	}{
		{
			"simple present",
			fooPath,
			common.MapStr{"foo": "bar"},
			"bar",
			true,
		},
		{
			"simple missing",
			fooPath,
			common.MapStr{},
			nil,
			false,
		},
		{
			"complex present",
			complexPath,
			common.MapStr{"foo": []interface{}{common.MapStr{"bar": []string{"bad", "good"}}}},
			"good",
			true,
		},
		{
			"complex missing",
			complexPath,
			common.MapStr{},
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValue, gotExists := tt.p.getFrom(tt.arg)
			if !reflect.DeepEqual(gotValue, tt.wantValue) {
				t.Errorf("path.getFrom() gotValue = %v, want %v", gotValue, tt.wantValue)
			}
			if gotExists != tt.wantExists {
				t.Errorf("path.getFrom() gotExists = %v, want %v", gotExists, tt.wantExists)
			}
		})
	}
}

func TestParsePath(t *testing.T) {
	tests := []struct {
		name    string
		arg     string
		wantP   path
		wantErr bool
	}{
		{
			"simple",
			"foo",
			path{}.extendMap("foo"),
			false,
		},
		{
			"complex",
			"foo.[0].bar.[1].baz",
			path{}.extendMap("foo").extendSlice(0).extendMap("bar").extendSlice(1).extendMap("baz"),
			false,
		},
		// TODO: The validation and testing for this needs to be better
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotP, err := parsePath(tt.arg)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotP, tt.wantP) {
				t.Errorf("parsePath() = %v, want %v", gotP, tt.wantP)
			}
		})
	}
}
