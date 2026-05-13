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

package add_host_metadata

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"

	"github.com/elastic/beats/v7/libbeat/processors/util"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/features"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-sysinfo/types"
)

var (
	hostName = "testHost"
	hostID   = "9C7FAB7B"
)

func TestConfigDefault(t *testing.T) {
	event := &beat.Event{
		Fields:    mapstr.M{},
		Timestamp: time.Now(),
	}
	testConfig, err := conf.NewConfigFrom(map[string]interface{}{})
	assert.NoError(t, err)

	p, err := New(testConfig, logptest.NewTestingLogger(t, ""))
	switch runtime.GOOS {
	case "windows", "darwin", "linux", "solaris":
		assert.NoError(t, err)
	default:
		assert.IsType(t, types.ErrNotImplemented, err)
		return
	}

	newEvent, err := p.Run(event)
	assert.NoError(t, err)

	v, err := newEvent.GetValue("host.os.family")
	assert.NoError(t, err)
	assert.NotNil(t, v)

	v, err = newEvent.GetValue("host.os.kernel")
	assert.NoError(t, err)
	assert.NotNil(t, v)

	v, err = newEvent.GetValue("host.os.name")
	assert.NoError(t, err)
	assert.NotNil(t, v)

	v, err = newEvent.GetValue("host.ip")
	assert.NoError(t, err)
	assert.NotNil(t, v)

	v, err = newEvent.GetValue("host.mac")
	assert.NoError(t, err)
	assert.NotNil(t, v)

	v, err = newEvent.GetValue("host.os.type")
	assert.NoError(t, err)
	assert.NotNil(t, v)
}

func TestConfigNetInfoDisabled(t *testing.T) {
	event := &beat.Event{
		Fields:    mapstr.M{},
		Timestamp: time.Now(),
	}
	testConfig, err := conf.NewConfigFrom(map[string]interface{}{
		"netinfo.enabled": false,
	})
	assert.NoError(t, err)

	p, err := New(testConfig, logptest.NewTestingLogger(t, ""))
	switch runtime.GOOS {
	case "windows", "darwin", "linux", "solaris":
		assert.NoError(t, err)
	default:
		assert.IsType(t, types.ErrNotImplemented, err)
		return
	}

	newEvent, err := p.Run(event)
	assert.NoError(t, err)

	v, err := newEvent.GetValue("host.os.family")
	assert.NoError(t, err)
	assert.NotNil(t, v)

	v, err = newEvent.GetValue("host.os.kernel")
	assert.NoError(t, err)
	assert.NotNil(t, v)

	v, err = newEvent.GetValue("host.os.name")
	assert.NoError(t, err)
	assert.NotNil(t, v)

	v, err = newEvent.GetValue("host.ip")
	assert.Error(t, err)
	assert.Nil(t, v)

	v, err = newEvent.GetValue("host.mac")
	assert.Error(t, err)
	assert.Nil(t, v)

	v, err = newEvent.GetValue("host.os.type")
	assert.NoError(t, err)
	assert.NotNil(t, v)
}

func TestConfigName(t *testing.T) {
	event := &beat.Event{
		Fields:    mapstr.M{},
		Timestamp: time.Now(),
	}

	config := map[string]interface{}{
		"name": "my-host",
	}

	testConfig, err := conf.NewConfigFrom(config)
	assert.NoError(t, err)

	p, err := New(testConfig, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	newEvent, err := p.Run(event)
	assert.NoError(t, err)

	for configKey, configValue := range config {
		t.Run(fmt.Sprintf("Check of %s", configKey), func(t *testing.T) {
			v, err := newEvent.GetValue(fmt.Sprintf("host.%s", configKey))
			assert.NoError(t, err)
			assert.Equal(t, configValue, v, "Could not find in %s", newEvent)
		})
	}
}

func TestConfigGeoEnabled(t *testing.T) {
	event := &beat.Event{
		Fields:    mapstr.M{},
		Timestamp: time.Now(),
	}

	config := map[string]interface{}{
		"geo.name":             "yerevan-am",
		"geo.location":         "40.177200, 44.503490",
		"geo.continent_name":   "Asia",
		"geo.country_name":     "Armenia",
		"geo.country_iso_code": "AM",
		"geo.region_name":      "Erevan",
		"geo.region_iso_code":  "AM-ER",
		"geo.city_name":        "Yerevan",
	}

	testConfig, err := conf.NewConfigFrom(config)
	assert.NoError(t, err)

	p, err := New(testConfig, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	newEvent, err := p.Run(event)
	assert.NoError(t, err)

	eventGeoField, err := newEvent.GetValue("host.geo")
	require.NoError(t, err)

	assert.Len(t, eventGeoField, len(config))
}

func TestConfigGeoDisabled(t *testing.T) {
	event := &beat.Event{
		Fields:    mapstr.M{},
		Timestamp: time.Now(),
	}

	config := map[string]interface{}{}

	testConfig, err := conf.NewConfigFrom(config)
	require.NoError(t, err)

	p, err := New(testConfig, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	newEvent, err := p.Run(event)

	require.NoError(t, err)

	eventGeoField, err := newEvent.GetValue("host.geo")
	assert.Error(t, err)
	assert.Nil(t, eventGeoField)
}

func TestEventWithReplaceFieldsFalse(t *testing.T) {
	cfg := map[string]interface{}{}
	cfg["replace_fields"] = false
	testConfig, err := conf.NewConfigFrom(cfg)
	assert.NoError(t, err)

	p, err := New(testConfig, logptest.NewTestingLogger(t, ""))
	switch runtime.GOOS {
	case "windows", "darwin", "linux", "solaris":
		assert.NoError(t, err)
	default:
		assert.IsType(t, types.ErrNotImplemented, err)
		return
	}

	cases := []struct {
		title                   string
		event                   beat.Event
		hostLengthLargerThanOne bool
		hostLengthEqualsToOne   bool
		expectedHostFieldLength int
	}{
		{
			"replace_fields=false with only host.name",
			beat.Event{
				Fields: mapstr.M{
					"host": mapstr.M{
						"name": hostName,
					},
				},
			},
			true,
			false,
			-1,
		},
		{
			"replace_fields=false with only host.id",
			beat.Event{
				Fields: mapstr.M{
					"host": mapstr.M{
						"id": hostID,
					},
				},
			},
			false,
			true,
			1,
		},
		{
			"replace_fields=false with host.name and host.id",
			beat.Event{
				Fields: mapstr.M{
					"host": mapstr.M{
						"name": hostName,
						"id":   hostID,
					},
				},
			},
			true,
			false,
			2,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			newEvent, err := p.Run(&c.event)
			assert.NoError(t, err)

			v, err := newEvent.GetValue("host")
			assert.NoError(t, err)
			assert.Equal(t, c.hostLengthLargerThanOne, len(v.(mapstr.M)) > 1) //nolint:errcheck // already checked
			assert.Equal(t, c.hostLengthEqualsToOne, len(v.(mapstr.M)) == 1)  //nolint:errcheck // already checked
			if c.expectedHostFieldLength != -1 {
				assert.Len(t, v.(mapstr.M), c.expectedHostFieldLength) //nolint:errcheck // already checked
			}
		})
	}
}

func TestEventWithReplaceFieldsTrue(t *testing.T) {
	cfg := map[string]interface{}{}
	cfg["replace_fields"] = true
	testConfig, err := conf.NewConfigFrom(cfg)
	assert.NoError(t, err)

	p, err := New(testConfig, logptest.NewTestingLogger(t, ""))
	switch runtime.GOOS {
	case "windows", "darwin", "linux", "solaris":
		assert.NoError(t, err)
	default:
		assert.IsType(t, types.ErrNotImplemented, err)
		return
	}

	cases := []struct {
		title                   string
		event                   beat.Event
		hostLengthLargerThanOne bool
		hostLengthEqualsToOne   bool
	}{
		{
			"replace_fields=true with host.name",
			beat.Event{
				Fields: mapstr.M{
					"host": mapstr.M{
						"name": hostName,
					},
				},
			},
			true,
			false,
		},
		{
			"replace_fields=true with host.id",
			beat.Event{
				Fields: mapstr.M{
					"host": mapstr.M{
						"id": hostID,
					},
				},
			},
			true,
			false,
		},
		{
			"replace_fields=true with host.name and host.id",
			beat.Event{
				Fields: mapstr.M{
					"host": mapstr.M{
						"name": hostName,
						"id":   hostID,
					},
				},
			},
			true,
			false,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			newEvent, err := p.Run(&c.event)
			assert.NoError(t, err)

			v, err := newEvent.GetValue("host")
			assert.NoError(t, err)
			assert.Equal(t, c.hostLengthLargerThanOne, len(v.(mapstr.M)) > 1) //nolint:errcheck // already checked
			assert.Equal(t, c.hostLengthEqualsToOne, len(v.(mapstr.M)) == 1)  //nolint:errcheck // already checked
		})
	}
}

func TestSkipAddingHostMetadata(t *testing.T) {
	hostIDMap := map[string]string{}
	hostIDMap["id"] = hostID

	hostNameMap := map[string]string{}
	hostNameMap["name"] = hostName

	hostIDNameMap := map[string]string{}
	hostIDNameMap["id"] = hostID
	hostIDNameMap["name"] = hostName

	cases := []struct {
		title        string
		event        beat.Event
		expectedSkip bool
	}{
		{
			"event only with host.name",
			beat.Event{
				Fields: mapstr.M{
					"host": mapstr.M{
						"name": hostName,
					},
				},
			},
			false,
		},
		{
			"event only with host.id",
			beat.Event{
				Fields: mapstr.M{
					"host": mapstr.M{
						"id": hostID,
					},
				},
			},
			true,
		},
		{
			"event with host.name and host.id",
			beat.Event{
				Fields: mapstr.M{
					"host": mapstr.M{
						"name": hostName,
						"id":   hostID,
					},
				},
			},
			true,
		},
		{
			"event without host field",
			beat.Event{
				Fields: mapstr.M{},
			},
			false,
		},
		{
			"event with field type map[string]string hostID",
			beat.Event{
				Fields: mapstr.M{
					"host": hostIDMap,
				},
			},
			true,
		},
		{
			"event with field type map[string]string host name",
			beat.Event{
				Fields: mapstr.M{
					"host": hostNameMap,
				},
			},
			false,
		},
		{
			"event with field type map[string]string host ID and name",
			beat.Event{
				Fields: mapstr.M{
					"host": hostIDNameMap,
				},
			},
			true,
		},
		{
			"event with field type string",
			beat.Event{
				Fields: mapstr.M{
					"host": "string",
				},
			},
			false,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			skip := skipAddingHostMetadata(&c.event)
			assert.Equal(t, c.expectedSkip, skip)
		})
	}
}

func TestFQDNEventSync(t *testing.T) {
	hostname := "hostname"
	fqdn := "fqdn"

	testConfig := conf.MustNewConfigFrom(map[string]interface{}{
		"cache.ttl": "5m",
	})

	// Start with FQDN off
	err := features.UpdateFromConfig(conf.MustNewConfigFrom(map[string]interface{}{
		"features.fqdn.enabled": false,
	}))
	require.NoError(t, err)

	p, err := New(testConfig, logptest.NewTestingLogger(t, ""))
	typedProc, ok := p.(*addHostMetadata)
	require.True(t, ok)
	typedProc.hostInfoFactory = func() (hostInfo, error) {
		return &mockHostInfo{
			Hostname: hostname,
			FQDN:     fqdn,
		}, nil
	}

	require.NoError(t, err)

	// update
	err = features.UpdateFromConfig(conf.MustNewConfigFrom(map[string]interface{}{
		"features.fqdn.enabled": true,
	}))
	require.NoError(t, err)

	t.Logf("updated FQDN")

	// run a number of events, make sure none have wrong hostname.
	assert.EventuallyWithT(t, func(collect *assert.CollectT) {
		resp, err := p.Run(&beat.Event{
			Fields: mapstr.M{},
		})
		require.NoError(collect, err)
		name, err := resp.Fields.GetValue("host.name")
		require.NoError(collect, err)
		assert.Equal(collect, fqdn, name)
	}, time.Second*3600, time.Millisecond*10)
}

func TestDataReload(t *testing.T) {
	var processingGoroutineCount int32 = 10
	testConfig := conf.MustNewConfigFrom(map[string]interface{}{
		"cache.ttl": "5m",
	})

	// Start with FQDN off
	err := features.UpdateFromConfig(conf.MustNewConfigFrom(map[string]interface{}{
		"features.fqdn.enabled": false,
	}))
	require.NoError(t, err)

	info := &mockHostInfo{}
	factory := func() (hostInfo, error) {
		return info, nil
	}

	p, err := newWithHostInfoFactory(testConfig, logptest.NewTestingLogger(t, ""), factory)
	require.NoError(t, err)

	// we should have a single data reload during creation
	assert.Equal(t, int64(1), info.HostInfoRequestCount.Load())
	assert.Equal(t, int64(0), info.FQDNRequestCount.Load())

	eventCount := atomic.Int32{} // this is used to ensure some events get processed before we do our assertions
	var finished atomic.Bool
	wg := &sync.WaitGroup{}
	t.Cleanup(func() {
		finished.Store(true)
		wg.Wait()
	})
	// start some goroutines enriching events in an infinite loop
	processEvents := func() {
		defer wg.Done()
		for !finished.Load() {
			_, err := p.Run(&beat.Event{
				Fields: mapstr.M{},
			})
			require.NoError(t, err)
			eventCount.Add(1)
		}
	}
	// Start several goroutines to call the processor in parallel
	for range processingGoroutineCount {
		wg.Add(1)
		go processEvents()
	}

	// Wait until at least some events have gone through.
	assert.EventuallyWithT(t, func(collect *assert.CollectT) {
		assert.Positive(collect, eventCount.Load())
	}, time.Second*5, time.Millisecond)

	// we should still have a single data reload since any requests should
	// use the cache until the FQDN flag changes.
	assert.Equal(t, int64(1), info.HostInfoRequestCount.Load())
	assert.Equal(t, int64(0), info.FQDNRequestCount.Load())

	// update
	err = features.UpdateFromConfig(conf.MustNewConfigFrom(map[string]interface{}{
		"features.fqdn.enabled": true,
	}))
	require.NoError(t, err)

	t.Logf("updated FQDN")

	// we should have reloaded the data once
	// note that with fqdn enabled, we still fetch the host info
	var previousEventCount = eventCount.Load()
	assert.EventuallyWithT(t, func(collect *assert.CollectT) {
		// Causality: there can be up to processingGoroutineCount pending
		// increments of eventCount from Run calls that already finished.
		// To guarantee that at least one run has happened since the
		// feature flag change, our event count must go up by _more_
		// than that.
		assert.Greater(collect, eventCount.Load(), previousEventCount+processingGoroutineCount)
	}, time.Second*5, time.Millisecond)

	// The FQDN flag has changed to true, there should be an additional host
	// info request for the FQDN case, as well as an FQDN request.
	assert.Equal(t, int64(2), info.HostInfoRequestCount.Load())
	assert.Equal(t, int64(1), info.FQDNRequestCount.Load())

	// update back to the original value
	err = features.UpdateFromConfig(conf.MustNewConfigFrom(map[string]interface{}{
		"features.fqdn.enabled": false,
	}))
	require.NoError(t, err)

	// we should have reloaded the data once more
	previousEventCount = eventCount.Load()
	assert.EventuallyWithT(t, func(collect *assert.CollectT) {
		assert.Greater(collect, eventCount.Load(), previousEventCount+processingGoroutineCount)
	}, time.Second*5, time.Millisecond)

	// Both values should be unchanged, including host info requests, because
	// it can still use the cached value from the original non-FQDN lookup.
	assert.Equal(t, int64(2), info.HostInfoRequestCount.Load())
	assert.Equal(t, int64(1), info.FQDNRequestCount.Load())
}

func TestFQDNLookup(t *testing.T) {
	hostname := "placeholder"

	tests := map[string]struct {
		fqdnLookupResult              string
		expectedHostName              string
		expectedFQDNLookupFailedCount int64
	}{
		"lookup_succeeds": {
			fqdnLookupResult:              "example.com",
			expectedHostName:              "example.com",
			expectedFQDNLookupFailedCount: 0,
		},
		"lookup_fails": {
			expectedHostName:              hostname,
			expectedFQDNLookupFailedCount: 1,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// Enable FQDN feature flag
			err := features.UpdateFromConfig(fqdnFeatureFlagConfig(true))
			require.NoError(t, err)
			defer func() {
				err = features.UpdateFromConfig(fqdnFeatureFlagConfig(true))
				require.NoError(t, err)
			}()

			// Create processor and check that FQDN lookup failed
			testConfig, err := conf.NewConfigFrom(map[string]interface{}{})
			require.NoError(t, err)

			factory := func() (hostInfo, error) {
				var fqdnError error
				if test.expectedFQDNLookupFailedCount > 0 {
					fqdnError = errors.New("hostname lookup failed")
				}
				return &mockHostInfo{
					Hostname: hostname,
					FQDN:     test.fqdnLookupResult,
					FQDNErr:  fqdnError,
				}, nil
			}
			p, err := newWithHostInfoFactory(testConfig, logptest.NewTestingLogger(t, ""), factory)
			require.NoError(t, err)

			addHostMetadataP, ok := p.(*addHostMetadata)
			require.True(t, ok)
			require.Equal(t, test.expectedFQDNLookupFailedCount, addHostMetadataP.metrics.FQDNLookupFailed.Get())
			// reset so next run is correct, registry is global
			addHostMetadataP.metrics.FQDNLookupFailed.Set(0)

			// Run event through processor and check that hostname reported
			// by processor is same as OS-reported hostname
			event := &beat.Event{
				Fields:    mapstr.M{},
				Timestamp: time.Now(),
			}
			newEvent, err := p.Run(event)
			require.NoError(t, err)

			v, err := newEvent.GetValue("host.name")
			require.NoError(t, err)
			require.Equal(t, test.expectedHostName, v)
		})
	}
}

func fqdnFeatureFlagConfig(fqdnEnabled bool) *conf.C {
	return conf.MustNewConfigFrom(map[string]interface{}{
		"features.fqdn.enabled": fqdnEnabled,
	})
}

type mockHostInfo struct {
	FQDN                 string
	Hostname             string
	FQDNErr              error
	FQDNRequestCount     atomic.Int64
	HostInfoRequestCount atomic.Int64
}

var _ hostInfo = &mockHostInfo{}

func (m *mockHostInfo) Info() types.HostInfo {
	m.HostInfoRequestCount.Add(1)
	return types.HostInfo{
		Hostname: m.Hostname,
		OS:       &types.OSInfo{},
	}
}

func (m *mockHostInfo) FQDNWithContext(_ context.Context) (string, error) {
	m.FQDNRequestCount.Add(1)
	if m.FQDNErr != nil {
		return "", m.FQDNErr
	}
	return m.FQDN, nil
}

// New constructs a new add_host_metadata processor with a custom host info factory
func newWithHostInfoFactory(cfg *conf.C, log *logp.Logger, factory hostInfoFactory) (beat.Processor, error) {
	c := defaultConfig()
	if err := cfg.Unpack(&c); err != nil {
		return nil, fmt.Errorf("fail to unpack the %v configuration: %w", processorName, err)
	}

	p := &addHostMetadata{
		config: c,
		caches: [2]hostMetadataCache{
			{data: mapstr.NewPointer(nil)},
			{data: mapstr.NewPointer(nil)},
		},
		logger: log.Named(logName),
		metrics: metrics{
			FQDNLookupFailed: monitoring.NewInt(reg, "fqdn_lookup_failed"),
		},
		hostInfoFactory: factory,
	}
	if _, err := p.loadData(features.FQDN()); err != nil {
		return nil, fmt.Errorf("failed to load data: %w", err)
	}

	if c.Geo != nil {
		geoFields, err := util.GeoConfigToMap(*c.Geo)
		if err != nil {
			return nil, err
		}
		p.geoData = mapstr.M{"host": mapstr.M{"geo": geoFields}}
	}

	return p, nil
}
