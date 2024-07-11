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
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	rd "github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

var (
	message  = "AUTH (redacted)"
	hostPort = fmt.Sprintf("%s:%s",
		getOrDefault(os.Getenv("REDIS_HOST"), "localhost"),
		getOrDefault(os.Getenv("REDIS_PORT"), "6379"))
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
	ec.events <- event
	return true
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
	logp.TestingSetup(logp.WithSelectors("redis input", "redis"))

	// Setup the input config.
	config := conf.MustNewConfigFrom(mapstr.M{
		"network":      "tcp",
		"type":         "redis",
		"hosts":        []string{hostPort},
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
	defer close(eventsCh)

	captor := newEventCaptor(eventsCh)
	defer captor.Close()

	connector := channel.ConnectorFunc(func(_ *conf.C, _ beat.ClientConfig) (channel.Outleter, error) {
		return channel.SubOutlet(captor), nil
	})

	// Mock the context.
	inputContext := input.Context{
		Done:     make(chan struct{}),
		BeatDone: make(chan struct{}),
	}

	// Setup the input
	input, err := NewInput(config, connector, inputContext)
	require.NoError(t, err)
	require.NotNil(t, input)

	// Run the input.
	input.Run()

	// Create Redis Client
	redisClient := createRedisClient(t)

	// Verify that event has been received
	verifiedCh := make(chan struct{})
	defer close(verifiedCh)

	emitInputData(t, verifiedCh, redisClient)

	event := <-eventsCh
	verifiedCh <- struct{}{}

	val, err := event.GetValue("message")
	require.NoError(t, err)
	require.Equal(t, message, val)
}

func emitInputData(t *testing.T, verifiedCh <-chan struct{}, pool *rd.Pool) {
	script := "local i = 0 for j=1,500000 do i = i + j end return i"

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		conn := pool.Get()
		defer conn.Close()

		for {
			select {
			case <-verifiedCh:
				return
			case <-ticker.C:
				_, err := conn.Do("EVAL", script, 0)
				require.NoError(t, err)
			}
		}
	}()
}

func createRedisClient(t *testing.T) *rd.Pool {
	idleTimeout := 60 * time.Second

	enabled := true

	tlsConfig, err := tlscommon.LoadTLSConfig(&tlscommon.Config{
		Enabled: &enabled,
		CAs:     []string{"_meta/certs/root-ca.pem"},
		Certificate: tlscommon.CertificateConfig{
			Certificate: "_meta/certs/server-cert.pem",
			Key:         "_meta/certs/server-key.pem",
		},
	})
	if err != nil {
		t.Fatalf("failed to load TLS configuration: %v", err)
	}

	return &rd.Pool{
		MaxIdle:     10,
		IdleTimeout: idleTimeout,
		Dial: func() (rd.Conn, error) {
			dialOptions := []rd.DialOption{
				// rd.DialUsername("username"),
				rd.DialPassword("password"),
				rd.DialConnectTimeout(idleTimeout),
				rd.DialReadTimeout(idleTimeout),
				rd.DialWriteTimeout(idleTimeout),
				rd.DialUseTLS(true),
				rd.DialTLSConfig(tlsConfig.ToConfig()),
			}

			c, err := rd.Dial("tcp", hostPort, dialOptions...)
			if err != nil {
				return nil, err
			}

			return c, err
		},
		TestOnBorrow: func(c rd.Conn, t time.Time) error {
			if time.Since(t) < idleTimeout {
				return nil
			}

			_, err := c.Do("PING")
			return err
		},
	}
}

func getOrDefault(s, defaultString string) string {
	if s == "" {
		return defaultString
	}
	return s
}
