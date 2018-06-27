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

// +build linux

package diskio

import (
	"testing"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/system"
)

func Test_Get_CLK_TCK(t *testing.T) {
	//usually the tick is 100
	assert.Equal(t, uint32(100), Get_CLK_TCK())
}

func TestDataNameFilter(t *testing.T) {
	oldFS := system.HostFS
	newFS := "_meta/testdata"
	system.HostFS = &newFS
	defer func() {
		system.HostFS = oldFS
	}()

	conf := map[string]interface{}{
		"module":                 "system",
		"metricsets":             []string{"diskio"},
		"diskio.include_devices": []string{"sda", "sda1", "sda2"},
	}

	f := mbtest.NewEventsFetcher(t, conf)

	if err := mbtest.WriteEvents(f, t); err != nil {
		t.Fatal("write", err)
	}

	data, err := f.Fetch()
	assert.NoError(t, err)
	assert.Equal(t, 3, len(data))
}

func TestDataEmptyFilter(t *testing.T) {
	oldFS := system.HostFS
	newFS := "_meta/testdata"
	system.HostFS = &newFS
	defer func() {
		system.HostFS = oldFS
	}()

	conf := map[string]interface{}{
		"module":     "system",
		"metricsets": []string{"diskio"},
	}

	f := mbtest.NewEventsFetcher(t, conf)

	if err := mbtest.WriteEvents(f, t); err != nil {
		t.Fatal("write", err)
	}

	data, err := f.Fetch()
	assert.NoError(t, err)
	assert.Equal(t, 10, len(data))
}
