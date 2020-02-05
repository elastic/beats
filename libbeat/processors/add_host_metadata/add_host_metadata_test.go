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
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/go-sysinfo/types"
)

func TestConfigDefault(t *testing.T) {
	event := &beat.Event{
		Fields:    common.MapStr{},
		Timestamp: time.Now(),
	}
	testConfig, err := common.NewConfigFrom(map[string]interface{}{})
	assert.NoError(t, err)

	p, err := New(testConfig)
	switch runtime.GOOS {
	case "windows", "darwin", "linux":
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
}

func TestConfigNetInfoEnabled(t *testing.T) {
	event := &beat.Event{
		Fields:    common.MapStr{},
		Timestamp: time.Now(),
	}
	testConfig, err := common.NewConfigFrom(map[string]interface{}{
		"netinfo.enabled": true,
	})
	assert.NoError(t, err)

	p, err := New(testConfig)
	switch runtime.GOOS {
	case "windows", "darwin", "linux":
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
}

func TestConfigName(t *testing.T) {
	event := &beat.Event{
		Fields:    common.MapStr{},
		Timestamp: time.Now(),
	}

	config := map[string]interface{}{
		"name": "my-host",
	}

	testConfig, err := common.NewConfigFrom(config)
	assert.NoError(t, err)

	p, err := New(testConfig)
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
		Fields:    common.MapStr{},
		Timestamp: time.Now(),
	}

	config := map[string]interface{}{
		"geo.name":             "yerevan-am",
		"geo.location":         "40.177200, 44.503490",
		"geo.continent_name":   "Asia",
		"geo.country_iso_code": "AM",
		"geo.region_name":      "Erevan",
		"geo.region_iso_code":  "AM-ER",
		"geo.city_name":        "Yerevan",
	}

	testConfig, err := common.NewConfigFrom(config)
	assert.NoError(t, err)

	p, err := New(testConfig)
	require.NoError(t, err)

	newEvent, err := p.Run(event)
	assert.NoError(t, err)

	eventGeoField, err := newEvent.GetValue("host.geo")
	require.NoError(t, err)

	assert.Len(t, eventGeoField, len(config))
}

func TestConfigGeoDisabled(t *testing.T) {
	event := &beat.Event{
		Fields:    common.MapStr{},
		Timestamp: time.Now(),
	}

	config := map[string]interface{}{}

	testConfig, err := common.NewConfigFrom(config)
	require.NoError(t, err)

	p, err := New(testConfig)
	require.NoError(t, err)

	newEvent, err := p.Run(event)

	require.NoError(t, err)

	eventGeoField, err := newEvent.GetValue("host.geo")
	assert.Error(t, err)
	assert.Equal(t, nil, eventGeoField)
}
