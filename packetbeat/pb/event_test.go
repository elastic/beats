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

package pb

import (
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/ecs"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestTimeMarshal(t *testing.T) {
	testTime := time.Now()
	f := NewFields()

	f.Process = &ecs.Process{
		Start: testTime,
		Parent: &ecs.Process{
			Start: testTime,
		},
	}

	m := mapstr.M{}
	err := f.MarshalMapStr(m)
	require.NoError(t, err)
	procData := m["process"]
	assert.Equal(t, testTime, procData.(mapstr.M)["start"])
	assert.Equal(t, testTime, procData.(mapstr.M)["parent"].(mapstr.M)["start"])

}

func TestPointerHandling(t *testing.T) {
	testInt := 10
	testStr := "test"
	// test to make to sure we correctly handle pointers that aren't structs
	// mostly checking to make sure we don't panic due to pointer/reflect bugs
	testStruct := struct {
		PointerInt       *int         `ecs:"one"`
		SecondPointerInt *int         `ecs:"two"`
		TestStruct       *ecs.Process `ecs:"struct"`
		StrPointer       *string      `ecs:"string"`
	}{
		PointerInt:       nil,
		SecondPointerInt: &testInt,
		StrPointer:       &testStr,
		TestStruct: &ecs.Process{
			Name: "Test",
		},
	}

	out := mapstr.M{}
	err := MarshalStruct(out, "test", testStruct)
	require.NoError(t, err)

	want := mapstr.M{
		"test": mapstr.M{
			"struct": mapstr.M{
				"name": "Test",
			},
			"two":    &testInt,
			"string": &testStr,
		},
	}

	require.Equal(t, want, out)
}

func TestMarshalMapStr(t *testing.T) {
	f := NewFields()
	f.Source = &ecs.Source{IP: "127.0.0.1"}
	// make sure recursion works properly
	f.Process = &ecs.Process{
		Parent: &ecs.Process{
			Name: "Foo",
			Parent: &ecs.Process{
				Name: "Bar",
			},
		},
	}

	m := mapstr.M{}
	if err := f.MarshalMapStr(m); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, mapstr.M{
		"event": mapstr.M{
			"kind":     "event",
			"category": []string{"network"},
			"type":     []string{"connection", "protocol"},
		},
		"source": mapstr.M{"ip": "127.0.0.1"},
		"process": mapstr.M{
			"parent": mapstr.M{
				"name": "Foo",
				"parent": mapstr.M{
					"name": "Bar",
				},
			},
		},
	}, m)
}

func TestComputeValues(t *testing.T) {
	f := Fields{
		Source:      &ecs.Source{IP: "127.0.0.1", Port: 4000, Bytes: 100},
		Destination: &ecs.Destination{IP: "127.0.0.2", Port: 80, Bytes: 200},
		Network:     ecs.Network{Transport: "tcp"},
	}

	localAddrs := []net.IP{net.ParseIP("127.0.0.1")}

	if err := f.ComputeValues(localAddrs, nil); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, f.Source.IP, f.Client.IP)
	assert.Equal(t, f.Destination.IP, f.Server.IP)
	assert.EqualValues(t, f.Network.Bytes, 300)
	assert.NotZero(t, f.Network.CommunityID)
	assert.Equal(t, f.Network.Type, "ipv4")
	assert.Equal(t, f.Network.Direction, "ingress")
}

func TestIsEmptyValue(t *testing.T) {
	assert.False(t, isEmptyValue(reflect.ValueOf(time.Duration(1))))
	assert.False(t, isEmptyValue(reflect.ValueOf(time.Duration(0))))
	assert.True(t, isEmptyValue(reflect.ValueOf(time.Duration(-1))))
}

func TestSkipFields(t *testing.T) {
	m := mapstr.M{}
	if err := MarshalStruct(m, "test", &struct {
		Field1 string `ecs:"field1"`
		Field2 string
		Field3 string `ecs:"field3"`
	}{
		Field1: "field1",
		Field2: "field2",
		Field3: "field3",
	}); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, mapstr.M{
		"test": mapstr.M{
			"field1": "field1",
			"field3": "field3",
		},
	}, m)
}
