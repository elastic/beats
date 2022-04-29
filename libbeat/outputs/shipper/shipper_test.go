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

package shipper

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestConvertMapStr(t *testing.T) {
	cases := []struct {
		name   string
		value  mapstr.M
		exp    *structpb.Value
		expErr string
	}{
		{
			name: "nil returns nil",
			exp:  structpb.NewNullValue(),
		},
		{
			name:  "empty map returns empty struct",
			value: mapstr.M{},
			exp:   protoStruct(t, nil),
		},
		{
			name: "returns error when type is not supported",
			value: mapstr.M{
				"key": time.Now(),
			},
			expErr: "invalid type: time.Time",
		},
		{
			name: "values are preserved",
			value: mapstr.M{
				"key1": "string",
				"key2": 42,
				"key3": 42.2,
				"key4": mapstr.M{
					"subkey1": "string",
					"subkey2": mapstr.M{
						"subsubkey1": "string",
					},
				},
			},
			exp: protoStruct(t, map[string]interface{}{
				"key1": "string",
				"key2": 42,
				"key3": 42.2,
				"key4": map[string]interface{}{
					"subkey1": "string",
					"subkey2": map[string]interface{}{
						"subsubkey1": "string",
					},
				},
			}),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			converted, err := convertMapStr(tc.value)
			if tc.expErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErr)
				require.Nil(t, converted)
				return
			}
			requireEqualProto(t, tc.exp, converted)
		})
	}
}

func protoStruct(t *testing.T, values map[string]interface{}) *structpb.Value {
	s, err := structpb.NewStruct(values)
	require.NoError(t, err)
	return structpb.NewStructValue(s)
}

func requireEqualProto(t *testing.T, expected, actual proto.Message) {
	require.True(
		t,
		proto.Equal(expected, actual),
		fmt.Sprintf("These two protobuf messages are not equal:\nexpected: %v\nactual:  %v", expected, actual),
	)
}
