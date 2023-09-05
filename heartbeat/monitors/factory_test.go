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

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-lookslike"

	"github.com/stretchr/testify/require"

	hbconfig "github.com/elastic/beats/v7/heartbeat/config"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/beat/events"
	"github.com/elastic/beats/v7/libbeat/common/fmtstr"
	"github.com/elastic/beats/v7/libbeat/processors/add_data_stream"
	"github.com/elastic/beats/v7/libbeat/processors/util"
)

var binfo = beat.Info{
	Beat:        "heartbeat",
	IndexPrefix: "heartbeat",
	Version:     "8.0.0",
}

func TestPreProcessors(t *testing.T) {
	tests := map[string]struct {
		settings           publishSettings
		location           *hbconfig.LocationWithID
		expectedIndex      string
		expectedDatastream *add_data_stream.DataStream
		monitorType        string
		wantIndexChange    bool
		wantErr            bool
	}{
		"no settings should yield no processor for lightweight monitor": {
			publishSettings{},
			nil,
			"",
			nil,
			"http",
			false,
			false,
		},
		"no settings should yield a data stream processor for browsers": {
			publishSettings{},
			nil,
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
		"exact index should be used exactly": {
			publishSettings{Index: *fmtstr.MustCompileEvent("test")},
			nil,
			"test",
			nil,
			"http",
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
			nil,
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
			nil,
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
		"with location": {
			publishSettings{},
			&hbconfig.LocationWithID{
				ID: "TestID",
				Geo: util.GeoConfig{
					Name:     "geo name",
					Location: "38.889722, -77.008889",
				},
			},
			"",
			nil,
			"http",
			false,
			false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			e := beat.Event{Meta: mapstr.M{}, Fields: mapstr.M{}}
			procs, err := preProcessors(binfo, tt.location, tt.settings, tt.monitorType)
			if tt.wantErr == true {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			expectedProcs := 1
			if tt.location != nil {
				expectedProcs++
			}
			if tt.wantIndexChange {
				expectedProcs++
			}

			require.Len(t, procs.List, expectedProcs)

			_, err = procs.Run(&e)

			t.Run("index name should be set", func(t *testing.T) {
				require.NoError(t, err)
				if tt.expectedIndex == "" {
					require.Nil(t, e.Meta[events.FieldMetaRawIndex])
				} else {
					require.Equal(t, tt.expectedIndex, e.Meta[events.FieldMetaRawIndex])
				}
			})

			if tt.location == nil {
				t.Run("observer location data should not be set", func(t *testing.T) {
					ok, _ := e.Fields.HasKey("observer")
					require.False(t, ok)
				})
			} else {
				t.Run("observer location data should be set", func(t *testing.T) {
					geoM, _ := util.GeoConfigToMap(tt.location.Geo)
					lookslike.MustCompile(map[string]interface{}{
						"observer": map[string]interface{}{
							"name": tt.location.ID,
							"geo":  geoM,
						},
					})
				})
			}

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
					dataStream, _ := dataStreamRaw.(add_data_stream.DataStream)
					require.Equal(t, eventDs, dataStream.Dataset, "event.dataset be identical to data_stream.dataset")

					require.Equal(t, *tt.expectedDatastream, dataStream)
				}
			})
		})
	}
}

func TestDisabledMonitor(t *testing.T) {
	testConfigs := []map[string]interface{}{
		{
			"type":     "test",
			"enabled":  "false",
			"schedule": "@every 10s",
		},
		{
			"streams": []map[string]interface{}{
				{
					"type":     "test",
					"enabled":  "false",
					"schedule": "@every 10s",
				},
			},
		},
	}

	for _, confMap := range testConfigs {
		conf, err := config.NewConfigFrom(confMap)
		require.NoError(t, err)

		reg, built, closed := mockPluginsReg()
		f, sched, fClose := makeMockFactory(reg)
		defer fClose()
		defer sched.Stop()
		runner, err := f.Create(&MockPipeline{}, conf)
		require.NoError(t, err)
		require.IsType(t, NoopRunner{}, runner)

		require.Equal(t, 0, built.Load())
		require.Equal(t, 0, closed.Load())
	}
}

func TestRunFrom(t *testing.T) {
	tests := []struct {
		name string
		loc  *hbconfig.LocationWithID
	}{
		{
			"no location",
			nil,
		},
		{
			"with id",
			&hbconfig.LocationWithID{
				ID: "test",
			},
		},
	}

	for _, tt := range tests {
		confMap := map[string]interface{}{
			"type":     "test",
			"urls":     []string{"http://example.net"},
			"schedule": "@every 1ms",
			"name":     "test",
		}
		if tt.loc != nil {
			geo, err := util.GeoConfigToMap(tt.loc.Geo)
			require.NoError(t, err)
			confMap["run_from"] = map[string]interface{}{
				"id":  tt.loc.ID,
				"geo": geo,
			}
		}

		conf, err := config.NewConfigFrom(confMap)
		require.NoError(t, err)

		reg, _, _ := mockPluginsReg()
		mockPipeline := &MockPipeline{}

		f, sched, fClose := makeMockFactory(reg)
		defer fClose()
		defer sched.Stop()

		makeTestMon := func() (*Monitor, error) {
			mIface, err := f.Create(mockPipeline, conf)
			if mIface == nil {
				return nil, err
			} else {
				return mIface.(*Monitor), err
			}
		}

		// Would fail if the previous newMonitor didn't free the monitor.id
		m1, m1Err := makeTestMon()
		require.NoError(t, m1Err)

		if tt.loc == nil {
			var emptyLoc *hbconfig.LocationWithID
			require.Equal(t, emptyLoc, m1.stdFields.RunFrom)
		} else {
			require.Equal(t, tt.loc, m1.stdFields.RunFrom)
		}
	}
}

func TestDuplicateMonitorIDs(t *testing.T) {
	serverMonConf := mockPluginConf(t, "custom", "custom", "@every 1ms", "http://example.net")
	badConf := mockBadPluginConf(t, "custom")
	reg, built, closed := mockPluginsReg()
	mockPipeline := &MockPipeline{}

	f, sched, fClose := makeMockFactory(reg)
	defer fClose()
	defer sched.Stop()

	makeTestMon := func() (*Monitor, error) {
		mIface, err := f.Create(mockPipeline, serverMonConf)
		if mIface == nil {
			return nil, err
		} else {
			return mIface.(*Monitor), err
		}
	}

	c, err := mockPipeline.Connect()
	require.NoError(t, err)

	// Ensure that an error is returned on a bad config
	_, m0Err := newMonitor(badConf, reg, c, sched.Add, nil, nil)
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
