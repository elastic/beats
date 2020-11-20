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

package monitorcfg

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common"
)

func TestAgentInput_ToStandardConfig(t *testing.T) {
	simpleStreamMap := common.MapStr{
		"name":     "fwae",
		"type":     "http",
		"urls":     "https://www.elastic.co",
		"schedule": "@every 10s",
		"data_stream": common.MapStr{
			"dataset": "synthetics.monitor",
			"type":    "logs",
		},
	}
	simpleStream, err := common.NewConfigFrom(simpleStreamMap)
	require.NoError(t, err)

	simpleStreamMapWithId := simpleStreamMap.Clone()
	simpleStreamMapWithId.Put("id", []string{"myId"})
	type fields struct {
		Id      string
		Name    string
		Meta    *common.Config
		Streams []*common.Config
	}
	tests := []struct {
		name    string
		fields  fields
		want    common.MapStr
		wantErr bool
	}{
		{
			"simple",
			fields{"myId", "", nil, []*common.Config{simpleStream}},
			simpleStreamMapWithId,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ai := AgentInput{
				Id:      tt.fields.Id,
				Name:    tt.fields.Name,
				Meta:    tt.fields.Meta,
				Streams: tt.fields.Streams,
			}
			got, err := ai.ToStandardConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("ToStandardConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			gotDbs := common.DebugString(got, false)
			wantCfg, err := common.NewConfigFrom(tt.want)
			require.NoError(t, err)
			wantDbs := common.DebugString(wantCfg, false)

			if !reflect.DeepEqual(gotDbs, wantDbs) {
				t.Errorf("ToStandardConfig() got = %v, want %v", gotDbs, wantDbs)
			}
		})
	}
}
