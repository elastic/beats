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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"runtime"

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

	p, err := newHostMetadataProcessor(testConfig)
	if runtime.GOOS != "windows" && runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		assert.IsType(t, types.ErrNotImplemented, err)
		return
	}
	assert.NoError(t, err)

	newEvent, err := p.Run(event)
	assert.NoError(t, err)

	v, err := newEvent.GetValue("host.os.family")
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

	p, err := newHostMetadataProcessor(testConfig)
	if runtime.GOOS != "windows" && runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		assert.IsType(t, types.ErrNotImplemented, err)
		return
	}
	assert.NoError(t, err)

	newEvent, err := p.Run(event)
	assert.NoError(t, err)

	v, err := newEvent.GetValue("host.os.family")
	assert.NoError(t, err)
	assert.NotNil(t, v)

	v, err = newEvent.GetValue("host.ip")
	assert.NoError(t, err)
	assert.NotNil(t, v)

	v, err = newEvent.GetValue("host.mac")
	assert.NoError(t, err)
	assert.NotNil(t, v)
}
