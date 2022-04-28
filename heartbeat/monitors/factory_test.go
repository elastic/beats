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
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/scheduler"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/beat/events"
	"github.com/elastic/beats/v7/libbeat/common/fmtstr"
	"github.com/elastic/beats/v7/libbeat/monitoring"
	"github.com/elastic/beats/v7/libbeat/processors/add_data_stream"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var binfo = beat.Info{
	Beat:        "heartbeat",
	IndexPrefix: "heartbeat",
	Version:     "8.0.0",
}

func TestPreProcessors(t *testing.T) {
	tests := map[string]struct {
		settings           publishSettings
		expectedIndex      string
		expectedDatastream *add_data_stream.DataStream
		monitorType        string
		wantProc           bool
		wantErr            bool
	}{
		"no settings should yield no processor": {
			publishSettings{},
			"",
			nil,
			"browser",
			false,
			false,
		},
		"exact index should be used exactly": {
			publishSettings{Index: *fmtstr.MustCompileEvent("test")},
			"test",
			nil,
			"browser",
			true,
			false,
		},
		"data stream should be type-namespace-dataset": {
			publishSettings{
				DataStream: &add_data_stream.DataStream{
					Namespace: "myNamespace",
					Dataset:   "myDataset",
					Type:      "myType",
				},
			},
			"myType-myDataset-myNamespace",
			&add_data_stream.DataStream{
				Namespace: "myNamespace",
				Dataset:   "myDataset",
				Type:      "myType",
			},
			"myType",
			true,
			false,
		},
		"data stream should use defaults": {
			publishSettings{
				DataStream: &add_data_stream.DataStream{},
			},
			"synthetics-browser-default",
			&add_data_stream.DataStream{
				Namespace: "default",
				Dataset:   "browser",
				Type:      "synthetics",
			},
			"browser",
			true,
			false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			e := beat.Event{Meta: mapstr.M{}, Fields: mapstr.M{}}
			procs, err := preProcessors(binfo, tt.settings, tt.monitorType)
			if tt.wantErr == true {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// If we're just setting event.dataset we only get the 1
			// otherwise we get a second add_data_stream processor
			if !tt.wantProc {
				require.Len(t, procs.List, 1)
				return
			}
			require.Len(t, procs.List, 2)

			_, err = procs.Run(&e)

			t.Run("index name should be set", func(t *testing.T) {
				require.NoError(t, err)
				require.Equal(t, tt.expectedIndex, e.Meta[events.FieldMetaRawIndex])
			})

			eventDs, err := e.GetValue("event.dataset")
			require.NoError(t, err)

			t.Run("event.dataset should always be present, preferring data_stream", func(t *testing.T) {
				dataset := tt.monitorType
				if tt.settings.DataStream != nil && tt.settings.DataStream.Dataset != "" {
					dataset = tt.settings.DataStream.Dataset
				}
				require.Equal(t, dataset, eventDs, "event.dataset be computed correctly")
				require.Regexp(t, regexp.MustCompile(`^.+`), eventDs, "should be a string > 1 char")
			})

			t.Run("event.data_stream", func(t *testing.T) {
				dataStreamRaw, _ := e.GetValue("data_stream")
				if tt.expectedDatastream != nil {
					dataStream := dataStreamRaw.(add_data_stream.DataStream)
					require.Equal(t, eventDs, dataStream.Dataset, "event.dataset be identical to data_stream.dataset")

					require.Equal(t, *tt.expectedDatastream, dataStream)
				}
			})
		})
	}
}

func TestDuplicateMonitorIDs(t *testing.T) {
	serverMonConf := mockPluginConf(t, "custom", "custom", "@every 1ms", "http://example.net")
	badConf := mockBadPluginConf(t, "custom", "@every 1ms")
	reg, built, closed := mockPluginsReg()
	pipelineConnector := &MockPipelineConnector{}

	sched := scheduler.Create(1, monitoring.NewRegistry(), time.Local, nil, false)
	defer sched.Stop()

	f := NewFactory(binfo, sched.Add, reg, false)
	makeTestMon := func() (*Monitor, error) {
		mIface, err := f.Create(pipelineConnector, serverMonConf)
		if mIface == nil {
			return nil, err
		} else {
			return mIface.(*Monitor), err
		}
	}

	// Ensure that an error is returned on a bad config
	_, m0Err := newMonitor(badConf, reg, pipelineConnector, sched.Add, nil, false)
	require.Error(t, m0Err)

	// Would fail if the previous newMonitor didn't free the monitor.id
	m1, m1Err := makeTestMon()
	require.NoError(t, m1Err)
	m1.Start()
	m2, m2Err := makeTestMon()
	require.NoError(t, m2Err)
	m2.Start()
	// Change the name so we can ensure that this is the currently active monitor
	m2.stdFields.Name = "mon2"
	// This used to trigger an error, but shouldn't any longer, we just log
	// the error, and ensure the last monitor wins
	require.NoError(t, m2Err)

	m, ok := f.byId[m2.stdFields.ID]
	require.True(t, ok)
	require.Equal(t, m2.stdFields.Name, m.stdFields.Name)
	m1.Stop()
	m2.Stop()

	// Two are counted as built. The bad config is missing a stdfield so it
	// doesn't complete construction
	require.Equal(t, 2, built.Load())
	// Only 2 closes, because the bad config isn't closed
	require.Equal(t, 2, closed.Load())
}
