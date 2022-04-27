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

//go:build integration
// +build integration

package redis

import (
	"testing"

	rd "github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/compose"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
)

const (
	password = "foobared"
)

func TestPasswords(t *testing.T) {
	t.Skip("Changing password affects other tests, see https://github.com/elastic/beats/v7/issues/10955")

	service := compose.EnsureUp(t, "redis")
	host := service.Host()

	registry := mb.NewRegister()
	err := registry.AddModule("redis", mb.DefaultModuleFactory)
	require.NoError(t, err)

	registry.MustAddMetricSet("redis", "test", newDummyMetricSet,
		mb.WithHostParser(parse.PassThruHostParser),
	)

	// Add password and ensure it gets reset
	defer func() {
		err := resetPassword(host, password)
		if err != nil {
			t.Fatal("resetting password", err)
		}
	}()

	err = addPassword(host, password)
	if err != nil {
		t.Fatal("adding password", err)
	}

	// Test Fetch metrics with missing password
	ms := getMetricSet(t, registry, getConfig(host, ""))
	_, err = ms.Connection().Do("PING")
	if assert.Error(t, err, "missing password") {
		assert.Contains(t, err, "NOAUTH Authentication required.")
	}

	// Config redis and metricset with an invalid password
	ms = getMetricSet(t, registry, getConfig(host, "blah"))
	_, err = ms.Connection().Do("PING")
	if assert.Error(t, err, "invalid password") {
		assert.Contains(t, err, "ERR invalid password")
	}

	// Config redis and metricset with a valid password
	ms = getMetricSet(t, registry, getConfig(host, password))
	_, err = ms.Connection().Do("PING")
	assert.Empty(t, err, "valid password")
}

// addPassword will add a password to redis.
func addPassword(host, pass string) error {
	c, err := rd.Dial("tcp", host)
	if err != nil {
		return err
	}
	defer c.Close()

	_, err = c.Do("CONFIG", "SET", "requirepass", pass)
	return err
}

// resetPassword changes the password to the redis DB.
func resetPassword(host, currentPass string) error {
	c, err := rd.Dial("tcp", host)
	if err != nil {
		return err
	}
	defer c.Close()

	_, err = c.Do("AUTH", currentPass)
	if err != nil {
		return err
	}

	_, err = c.Do("CONFIG", "SET", "requirepass", "")
	return err
}

// dummyMetricSet is a metricset used only to instantiate a metricset
// from config using a registry
type dummyMetricSet struct {
	*MetricSet
}

func newDummyMetricSet(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := NewMetricSet(base)
	return &dummyMetricSet{ms}, err
}

func (m *dummyMetricSet) Fetch(r mb.ReporterV2) {
}

func getMetricSet(t *testing.T, registry *mb.Register, config map[string]interface{}) *MetricSet {
	t.Helper()

	c, err := conf.NewConfigFrom(config)
	require.NoError(t, err)

	_, metricsets, err := mb.NewModule(c, registry)
	require.NoError(t, err)
	require.Len(t, metricsets, 1)

	ms, ok := metricsets[0].(*dummyMetricSet)
	require.True(t, ok, "metricset must be dummyMetricSet")

	return ms.MetricSet
}

func getConfig(host string, password string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "redis",
		"metricsets": "test",
		"hosts":      []string{host},
		"password":   password,
	}
}
