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

package monitors

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/beat/events"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/fmtstr"
)

func TestSetupIndexProcessor(t *testing.T) {
	binfo := beat.Info{
		Beat:        "heartbeat",
		IndexPrefix: "heartbeat",
		Version:     "8.0.0",
	}
	tests := []struct {
		name          string
		settings      publishSettings
		expectedIndex string
		wantProc      bool
		wantErr       bool
	}{
		{
			"no settings should yield no processor",
			publishSettings{},
			"",
			false,
			false,
		},
		{
			"exact index should be used exactly",
			publishSettings{Index: *fmtstr.MustCompileEvent("test")},
			"test",
			true,
			false,
		},
		{
			"data stream should be type-namespace-dataset",
			publishSettings{
				DataStream: &datastream{
					Type:      "myType",
					Dataset:   "myDataset",
					Namespace: "myNamespace",
				},
			},
			"myType-myDataset-myNamespace",
			true,
			false,
		},
		{
			"data stream should use defaults",
			publishSettings{
				DataStream: &datastream{},
			},
			"synthetics-generic-default",
			true,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := beat.Event{Meta: common.MapStr{}, Fields: common.MapStr{}}
			proc, err := setupIndexProcessor(binfo, tt.settings)
			if tt.wantErr == true {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			if !tt.wantProc {
				require.Nil(t, proc)
				return
			}

			_, err = proc.Run(&e)
			require.Equal(t, tt.expectedIndex, e.Meta[events.FieldMetaRawIndex])
		})
	}
}
