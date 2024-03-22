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

package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-autodiscover/bus"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-ucfg"

	"github.com/elastic/beats/v7/libbeat/autodiscover"
)

func TestConfigWithCustomBuilders(t *testing.T) {
	autodiscover.Registry.AddBuilder("mock", newMockBuilder)

	cfg := mapstr.M{
		"hints.enabled": false,
		"builders": []mapstr.M{
			{
				"mock": mapstr.M{},
			},
		},
	}

	config := conf.MustNewConfigFrom(&cfg)
	c := defaultConfig()
	err := config.Unpack(&c)
	assert.NoError(t, err)

	cfg1 := mapstr.M{
		"hints.enabled": false,
	}
	config, err = conf.NewConfigFrom(&cfg1)
	c = defaultConfig()
	err = config.Unpack(&c)
	assert.Error(t, err)
}

func TestConfigWithIncorrectScope(t *testing.T) {
	cfg := mapstr.M{
		"scope":         "node",
		"resource":      "service",
		"hints.enabled": true,
	}

	config := conf.MustNewConfigFrom(&cfg)
	c := defaultConfig()
	err := config.Unpack(&c)
	assert.NoError(t, err)

	assert.Equal(t, "service", c.Resource)
	assert.Equal(t, "cluster", c.Scope)
}

func TestConfigLeaseFields(t *testing.T) {
	cfg := mapstr.M{
		"scope":  "cluster",
		"unique": "true",
	}

	tests := []struct {
		LeaseDuration string
		RenewDeadline string
		RetryPeriod   string
		message       string
	}{
		{
			LeaseDuration: "20seconds",
			RenewDeadline: "15s",
			RetryPeriod:   "2s",
			message:       "incorrect lease duration, should be set to default",
		},
		{
			LeaseDuration: "20s",
			RenewDeadline: "15minutes",
			RetryPeriod:   "2s",
			message:       "incorrect renew deadline, should be set to default",
		},
		{
			LeaseDuration: "20s",
			RenewDeadline: "15s",
			RetryPeriod:   "2hrs",
			message:       "incorrect retry period, should be set to default",
		},
	}

	for _, test := range tests {
		cfg["leader_leaseduration"] = test.LeaseDuration
		cfg["leader_renewdeadline"] = test.RenewDeadline
		cfg["leader_retryperiod"] = test.RetryPeriod

		config := conf.MustNewConfigFrom(&cfg)

		c := defaultConfig()
		err := config.Unpack(&c)
		assert.Errorf(t, err, test.message)
	}
}

type mockBuilder struct {
}

func newMockBuilder(_ *conf.C) (autodiscover.Builder, error) {
	return &mockBuilder{}, nil
}

func (m *mockBuilder) CreateConfig(event bus.Event, options ...ucfg.Option) []*conf.C {
	return nil
}
