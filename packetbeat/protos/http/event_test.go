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

package http

import (
	"reflect"
	"testing"

	"github.com/elastic/ecs/code/go/ecs"
)

// TestProtocolFieldsIsInSyncWithECS ensures that Packetbeat's clone of
// ecs.Http stays in sync.
func TestProtocolFieldsIsInSyncWithECS(t *testing.T) {
	ecs := getFields(reflect.TypeOf(ecs.Http{}))
	packetbeat := getFields(reflect.TypeOf(ProtocolFields{}))

	for name := range ecs {
		_, found := packetbeat[name]
		if !found {
			t.Errorf("Packetbeat is missing field=%v that's defined in ECS HTTP", name)
		}
		delete(packetbeat, name)
	}

	for name := range packetbeat {
		t.Errorf("packetbeat has more HTTP fields than ECS: %v", name)
	}
}

func getFields(typ reflect.Type) map[string]reflect.Type {
	fields := map[string]reflect.Type{}
	for i := 0; i < typ.NumField(); i++ {
		structField := typ.Field(i)
		tag := structField.Tag.Get("ecs")
		if tag == "" {
			continue
		}

		fields[tag] = structField.Type
	}
	return fields
}
