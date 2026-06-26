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

package redis

import (
	"context"
	"sync"
	"testing"
	"time"

	rd "github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/filebeat/input/redis/testutil"
	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

var (
	message = "AUTH (redacted)"
)

type eventCaptor struct {
	c         chan struct{}
	closeOnce sync.Once
	closed    bool
	events    chan beat.Event
}

func newEventCaptor(events chan beat.Event) channel.Outleter {
	return &eventCaptor{
		c:      make(chan struct{}),
		events: events,
	}
}

func (ec *eventCaptor) OnEvent(event beat.Event) bool {
	select {
	case ec.events <- event:
		return true
	case <-ec.c:
		// captor has been closed
		return false
	}
}

func (ec *eventCaptor) Close() error {
	ec.closeOnce.Do(func() {
		ec.closed = true
		close(ec.c)
	})
	return nil
}

func (ec *eventCaptor) Done() <-chan struct{} {
	return ec.c
}

func TestInput(t *testing.T) {
	// Setup the input config.
	config := conf.MustNewConfigFrom(mapstr.M{
		"network":      "tcp",
		"type":         "redis",
		"hosts":        []string{testutil.TLSHostPort()},
		"password":     "password",
		"maxconn":      10,
		"idle_timeout": 60 * time.Second,
		"ssl": mapstr.M{
			"enabled":                 true,
			"certificate_authorities": []string{"_meta/certs/root-ca.pem"},
			"certificate":             "_meta/certs/server-cert.pem",
			"key":                     "_meta/certs/server-key.pem",
		},
	})

	// Route input events through our captor instead of sending through ES.
	eventsCh := make(chan beat.Event)

	captor := newEventCaptor(eventsCh)

	connector := channel.ConnectorFunc(func(_ *conf.C, _ beat.ClientConfig) (channel.Outleter, error) {
		return channel.SubOutlet(captor), nil
	})

	// Mock the context.
	inputContext := input.Context{
		Done:     make(chan struct{}),
		BeatDone: make(chan struct{}),
	}

	logger := logptest.NewTestingLogger(t, "")
	// Setup the input
	input, err := NewInput(config, connector, inputContext, logger)
	require.NoError(t, err)
	require.NotNil(t, input)

	t.Cleanup(func() {
		// Captor must close before input during cleanup, otherwise it can keep
		// trying to send to eventsCh after we stop reading it, which blocks
		// and prevents input.Stop() from returning since it waits for the
		// harvester to close.
		captor.Close()
		input.Stop()
	})

	// Run the input.
	input.Run()

	// Create Redis Client
	redisClient := testutil.CreateTLSClient(t, "password")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testutil.EmitInputData(t, ctx, redisClient)

	select {
	case event := <-eventsCh:
		val, err := event.GetValue("message")
		require.NoError(t, err)
		require.Equal(t, message, val)
		val, err = event.GetValue("redis")
		require.NoError(t, err)
		role := val.(mapstr.M)["slowlog"].(mapstr.M)["role"] //nolint:errcheck //Safe to ignore in tests
		require.Equal(t, "master", role)
	case <-time.After(30 * time.Second):
		t.Fatal("Timeout waiting for event")
	}
}

func TestAuthenticate(t *testing.T) {
	var redisConfig config
	var pool *rd.Pool
	var conn rd.Conn
	var err error

	tlsHostPort := testutil.TLSHostPort()

	redisConfig = createRedisConfig("", "password")
	require.NotEmpty(t, redisConfig, "redisConfig should not be empty")
	pool = CreatePool(tlsHostPort, redisConfig)
	conn, err = pool.Dial()
	require.NoError(t, err)
	_, err = conn.Do("PING")
	require.NoError(t, err)

	redisConfig = createRedisConfig("", "password1")
	require.NotEmpty(t, redisConfig, "redisConfig should not be empty")
	pool = CreatePool(tlsHostPort, redisConfig)
	_, err = pool.Dial()
	require.Error(t, err)
	require.Equal(t, rd.Error("WRONGPASS invalid username-password pair or user is disabled."), err)

	redisConfig = createRedisConfig("testuser", "testpass")
	require.NotEmpty(t, redisConfig, "redisConfig should not be empty")
	pool = CreatePool(tlsHostPort, redisConfig)
	conn, err = pool.Dial()
	require.NoError(t, err)
	_, err = conn.Do("PING")
	require.NoError(t, err)

	redisConfig = createRedisConfig("testuser", "testpass1")
	require.NotEmpty(t, redisConfig, "redisConfig should not be empty")
	pool = CreatePool(tlsHostPort, redisConfig)
	_, err = pool.Dial()
	require.Error(t, err)
	require.Equal(t, rd.Error("WRONGPASS invalid username-password pair or user is disabled."), err)
}

func createRedisConfig(username string, password string) config {
	cfg := conf.MustNewConfigFrom(mapstr.M{
		"network":      "tcp",
		"type":         "redis",
		"hosts":        []string{testutil.TLSHostPort()},
		"maxconn":      10,
		"idle_timeout": 60 * time.Second,
		"ssl": mapstr.M{
			"enabled":                 true,
			"certificate_authorities": []string{"_meta/certs/root-ca.pem"},
			"certificate":             "_meta/certs/server-cert.pem",
			"key":                     "_meta/certs/server-key.pem",
		},
	})

	redisConfig := defaultConfig()
	err := cfg.Unpack(&redisConfig)
	if err != nil {
		return config{}
	}

	if username != "" {
		redisConfig.Username = username
	}

	if password != "" {
		redisConfig.Password = password
	}

	if redisConfig.TLS.IsEnabled() {
		tlsConfig, _ := tlscommon.LoadTLSConfig(redisConfig.TLS, logp.NewNopLogger())
		redisConfig.tlsConfig = tlsConfig.ToConfig()
	}

	return redisConfig
}
