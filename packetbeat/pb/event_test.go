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

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/ecs"
)

func TestMarshalMapStr(t *testing.T) {
	f := NewFields()
	f.Source = &ecs.Source{IP: "127.0.0.1"}

	m := common.MapStr{}
	if err := f.MarshalMapStr(m); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, common.MapStr{
		"event": common.MapStr{
			"kind":     "event",
			"category": []string{"network"},
			"type":     []string{"connection", "protocol"},
		},
		"source": common.MapStr{"ip": "127.0.0.1"},
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
	m := common.MapStr{}
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
	assert.Equal(t, common.MapStr{
		"test": common.MapStr{
			"field1": "field1",
			"field3": "field3",
		},
	}, m)
}
