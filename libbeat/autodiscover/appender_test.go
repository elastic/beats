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

package autodiscover

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/bus"
)

type fakeAppender struct{}

func (f *fakeAppender) Append(event bus.Event) {
	event["foo"] = "bar"
}

func newFakeAppender(_ *common.Config) (Appender, error) {
	return &fakeAppender{}, nil
}

func TestAppenderRegistry(t *testing.T) {
	// Add a new builder
	reg := NewRegistry()
	reg.AddAppender("fake", newFakeAppender)

	// Check if that appender is available in registry
	b := reg.GetAppender("fake")
	assert.NotNil(t, b)

	// Generate a config with type fake
	config := AppenderConfig{
		Type: "fake",
	}

	cfg, err := common.NewConfigFrom(&config)

	// Make sure that config building doesn't fail
	assert.Nil(t, err)
	appender, err := reg.BuildAppender(cfg)
	assert.Nil(t, err)
	assert.NotNil(t, appender)

	// Attempt to build using an array of configs
	Registry.AddAppender("fake", newFakeAppender)
	cfgs := []*common.Config{cfg}
	appenders, err := NewAppenders(cfgs)
	assert.Nil(t, err)
	assert.Equal(t, len(appenders), 1)

	// Attempt to build using an incorrect config
	incorrectConfig := AppenderConfig{
		Type: "wrong",
	}
	icfg, err := common.NewConfigFrom(&incorrectConfig)
	assert.Nil(t, err)
	cfgs = append(cfgs, icfg)
	appenders, err = NewAppenders(cfgs)
	assert.NotNil(t, err)
	assert.Nil(t, appenders)

	// Try to append onto an event using fakeAppender and the result should have one item
	event := bus.Event{}
	appender.Append(event)
	assert.Equal(t, len(event), 1)
	assert.Equal(t, event["foo"], "bar")

	appenders = Appenders{}
	appenders = append(appenders, appender)

	// Try using appenders object for the same as above and expect
	// the same result
	event = bus.Event{}
	appenders.Append(event)
	assert.Equal(t, len(event), 1)
	assert.Equal(t, event["foo"], "bar")
}
