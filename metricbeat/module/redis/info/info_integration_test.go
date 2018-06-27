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

// +build integration

package info

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/redis"

	rd "github.com/garyburd/redigo/redis"
	"github.com/stretchr/testify/assert"
)

const (
	password = "foobared"
)

var redisHost = redis.GetRedisEnvHost() + ":" + redis.GetRedisEnvPort()

func TestFetch(t *testing.T) {
	compose.EnsureUp(t, "redis")

	f := mbtest.NewEventFetcher(t, getConfig(""))
	event, err := f.Fetch()
	if err != nil {
		t.Fatal("fetch", err)
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)

	// Check fields
	assert.Equal(t, 8, len(event))
	server := event["server"].(common.MapStr)
	assert.Equal(t, "standalone", server["mode"])
}

func TestData(t *testing.T) {
	compose.EnsureUp(t, "redis")

	f := mbtest.NewEventFetcher(t, getConfig(""))

	err := mbtest.WriteEvent(f, t)
	if err != nil {
		t.Fatal("write", err)
	}
}

func TestPasswords(t *testing.T) {
	compose.EnsureUp(t, "redis")

	// Add password and ensure it gets reset
	defer func() {
		err := resetPassword(redisHost, password)
		if err != nil {
			t.Fatal("resetting password", err)
		}
	}()

	err := addPassword(redisHost, password)
	if err != nil {
		t.Fatal("adding password", err)
	}

	// Test Fetch metrics with missing password
	f := mbtest.NewEventFetcher(t, getConfig(""))
	_, err = f.Fetch()
	if assert.Error(t, err, "missing password") {
		assert.Contains(t, err, "NOAUTH Authentication required.")
	}

	// Config redis and metricset with an invalid password
	f = mbtest.NewEventFetcher(t, getConfig("blah"))
	_, err = f.Fetch()
	if assert.Error(t, err, "invalid password") {
		assert.Contains(t, err, "ERR invalid password")
	}

	// Config redis and metricset with a valid password
	f = mbtest.NewEventFetcher(t, getConfig(password))
	_, err = f.Fetch()
	assert.NoError(t, err, "valid password")
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

// writeToRedis will write to the default DB 0.
func writeToRedis(host string) error {
	c, err := rd.Dial("tcp", host)
	if err != nil {
		return err
	}
	defer c.Close()

	_, err = c.Do("SET", "foo", "bar")
	return err
}

func getConfig(password string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "redis",
		"metricsets": []string{"info"},
		"hosts":      []string{redisHost},
		"password":   password,
	}
}
