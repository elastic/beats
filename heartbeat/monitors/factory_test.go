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
	"github.com/elastic/beats/v7/libbeat/processors/add_data_stream_index"
)

func TestSetupIndexProcessor(t *testing.T) {
	binfo := beat.Info{
		Beat:        "heartbeat",
		IndexPrefix: "heartbeat",
		Version:     "8.0.0",
	}
	tests := map[string]struct {
		settings      publishSettings
		expectedIndex string
		monitorType   string
		wantProc      bool
		wantErr       bool
	}{
		"no settings should yield no processor": {
			publishSettings{},
			"",
			"browser",
			false,
			false,
		},
		"exact index should be used exactly": {
			publishSettings{Index: *fmtstr.MustCompileEvent("test")},
			"test",
			"browser",
			true,
			false,
		},
		"data stream should be type-namespace-dataset": {
			publishSettings{
				DataStream: &add_data_stream_index.DataStream{
					Namespace: "myNamespace",
					Dataset:   "myDataset",
					Type:      "myType",
				},
			},
			"myType-myDataset-myNamespace",
			"myType",
			true,
			false,
		},
		"data stream should use defaults": {
			publishSettings{
				DataStream: &add_data_stream_index.DataStream{},
			},
			"synthetics-browser-default",
			"browser",
			true,
			false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			e := beat.Event{Meta: common.MapStr{}, Fields: common.MapStr{}}
			proc, err := setupIndexProcessor(binfo, tt.settings, tt.monitorType)
			if tt.wantErr == true {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			if !tt.wantProc {
				require.Nil(t, proc)
				return
			}

			require.NotNil(t, proc)
			_, err = proc.Run(&e)
			require.Equal(t, tt.expectedIndex, e.Meta[events.FieldMetaRawIndex])
		})
	}
}
