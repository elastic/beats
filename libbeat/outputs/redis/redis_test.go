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

package redis

import (
	"context"
	"errors"
	"testing"

	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/codec"
	"github.com/elastic/beats/v7/libbeat/outputs/codec/json"
	_ "github.com/elastic/beats/v7/libbeat/outputs/codec/json"
	"github.com/elastic/beats/v7/libbeat/outputs/outest"
	"github.com/elastic/beats/v7/libbeat/outputs/outil"
	"github.com/elastic/beats/v7/libbeat/outputs/outputtest"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

type checker func(*testing.T, outputs.Group)

func checks(cs ...checker) checker {
	return func(t *testing.T, g outputs.Group) {
		for _, c := range cs {
			c(t, g)
		}
	}
}

func clientsLen(required int) checker {
	return func(t *testing.T, group outputs.Group) {
		assert.Len(t, group.Clients, required)
	}
}

func clientPassword(index int, pass string) checker {
	return func(t *testing.T, group outputs.Group) {
		redisClient := group.Clients[index].(*backoffClient) //nolint:errcheck //This is a test file, can ignore
		assert.Equal(t, redisClient.client.password, pass)
	}
}

func TestMakeRedis(t *testing.T) {
	tests := map[string]struct {
		config map[string]interface{}
		valid  bool
		checks checker
	}{
		"no host": {
			config: map[string]interface{}{
				"hosts": []string{},
			},
		},
		"invald scheme": {
			config: map[string]interface{}{
				"hosts": []string{"redisss://localhost:6379"},
			},
		},
		"Single host": {
			config: map[string]interface{}{
				"hosts": []string{"localhost:6379"},
			},
			valid:  true,
			checks: checks(clientsLen(1), clientPassword(0, "")),
		},
		"Multiple hosts": {
			config: map[string]interface{}{
				"hosts": []string{"redis://localhost:6379", "rediss://localhost:6380"},
			},
			valid:  true,
			checks: clientsLen(2),
		},
		"Default password": {
			config: map[string]interface{}{
				"hosts":    []string{"redis://localhost:6379"},
				"password": "defaultPassword",
			},
			valid:  true,
			checks: checks(clientsLen(1), clientPassword(0, "defaultPassword")),
		},
		"Specific and default password": {
			config: map[string]interface{}{
				"hosts":    []string{"redis://localhost:6379", "rediss://:mypassword@localhost:6380"},
				"password": "defaultPassword",
			},
			valid: true,
			checks: checks(
				clientsLen(2),
				clientPassword(0, "defaultPassword"),
				clientPassword(1, "mypassword"),
			),
		},
	}
	beatInfo := beat.Info{Beat: "libbeat", Version: "1.2.3"}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			logger := logptest.NewTestingLogger(t, "")
			beatInfo.Logger = logger
			cfg, err := config.NewConfigFrom(test.config)
			assert.NoError(t, err)
			groups, err := makeRedis(nil, beatInfo, outputs.NewNilObserver(), cfg)
			assert.Equal(t, err == nil, test.valid)
			if err != nil && test.valid {
				t.Log(err)
			}
			if test.checks != nil {
				test.checks(t, groups)
			}
		})
	}
}

func TestKeySelection(t *testing.T) {
	cases := map[string]struct {
		cfg   map[string]interface{}
		event beat.Event
		want  string
	}{
		"key configured": {
			cfg:  map[string]interface{}{"key": "test"},
			want: "test",
		},
		"key must keep case": {
			cfg:  map[string]interface{}{"key": "Test"},
			want: "Test",
		},
		"key setting": {
			cfg: map[string]interface{}{
				"keys": []map[string]interface{}{{"key": "test"}},
			},
			want: "test",
		},
		"keys setting must keep case": {
			cfg: map[string]interface{}{
				"keys": []map[string]interface{}{{"key": "Test"}},
			},
			want: "Test",
		},
		"use event field": {
			cfg: map[string]interface{}{"key": "test-%{[field]}"},
			event: beat.Event{
				Fields: mapstr.M{"field": "from-event"},
			},
			want: "test-from-event",
		},
		"use event field must keep case": {
			cfg: map[string]interface{}{"key": "Test-%{[field]}"},
			event: beat.Event{
				Fields: mapstr.M{"field": "From-Event"},
			},
			want: "Test-From-Event",
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			selector, err := buildKeySelector(config.MustNewConfigFrom(test.cfg))
			if err != nil {
				t.Fatalf("Failed to parse configuration: %v", err)
			}

			got, err := selector.Select(&test.event)
			if err != nil {
				t.Fatalf("Failed to create key name: %v", err)
			}

			if test.want != got {
				t.Errorf("Pipeline name missmatch (want: %v, got: %v)", test.want, got)
			}
		})
	}
}

func TestClientOutputListener(t *testing.T) {
	type publishCase struct {
		assertFn func(*testing.T, error)
		events   []beat.Event
	}
	type testCase struct {
		name            string
		publishCases    []publishCase
		makePublishFn   func(c *client, conn redis.Conn) publishFn
		mockSetup       func(conn *redisMock)
		events          []beat.Event
		expectedMetrics outputtest.Metrics
	}

	logger := logptest.NewTestingLogger(t, "",
		// only print stacktrace for errors above ErrorLevel.
		zap.AddStacktrace(zapcore.ErrorLevel+1))

	baseCfgMap := map[string]interface{}{
		"hosts":    []string{"localhost:6379"},
		"key":      "test",
		"datatype": "list",
	}

	testCases := []testCase{
		{
			name: "publishEventsPipeline",
			publishCases: []publishCase{
				{assertFn: func(t *testing.T, err error) {
					// as not all events succeed, Publish returns an error
					require.Error(t, err, "call to Publish return an error")
				},
					events: []beat.Event{
						{Fields: mapstr.M{"message": "event 1", "outcome": "success"}},
						{Fields: mapstr.M{"message": "event 2", "outcome": "retry"}},
						{Fields: mapstr.M{"message": "event 3", "outcome": "key-fail"}},
						{Fields: mapstr.M{"message": "event 4", "outcome": "encode-fail"}}},
				},
			},
			makePublishFn: func(c *client, conn redis.Conn) publishFn {
				return c.publishEventsPipeline(conn, "")
			},
			mockSetup: func(conn *redisMock) {
				conn.receiveRet = []error{nil, errors.New("pipeline: 2nd event fails")}
			},
			expectedMetrics: outputtest.Metrics{
				Total:     4,
				Acked:     1,
				Dropped:   2,
				Retryable: 1,
				Batches:   1,
			},
		},
		{
			name: "publishEventsBulk",
			publishCases: []publishCase{
				{
					assertFn: func(t *testing.T, err error) {
						require.NoError(t, err, "1st call to Publish should succeed")
					},
					events: []beat.Event{
						{Fields: mapstr.M{"message": "event 1", "outcome": "success"}}},
				},
				{
					assertFn: func(t *testing.T, err error) {
						require.Error(t, err, "2nd call to Publish should error")
					},
					events: []beat.Event{
						{Fields: mapstr.M{"message": "event 2", "outcome": "Do error"}}},
				},
			},
			makePublishFn: func(c *client, conn redis.Conn) publishFn {
				return c.publishEventsBulk(conn, "")
			},
			mockSetup: func(conn *redisMock) {
				conn.doRet = []error{nil, errors.New("do error")}
			},
			events: []beat.Event{
				{Fields: mapstr.M{"message": "event 1", "outcome": "success"}}},
			expectedMetrics: outputtest.Metrics{
				Total:     2,
				Acked:     1,
				Retryable: 1,
				Batches:   2,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, _ := config.NewConfigFrom(baseCfgMap)
			reg := monitoring.NewRegistry()
			group, err := makeRedis(nil,
				beat.Info{
					Beat:   "TestMetricsFastPath",
					Logger: logger},
				outputs.NewStats(reg),
				cfg)
			require.NoError(t, err)
			require.Len(t, group.Clients, 1)

			counter := &beat.CountOutputListener{}
			listener := publisher.OutputListener{Listener: counter}

			bc := group.Clients[0].(*backoffClient)
			c := bc.client
			c.key = outil.MakeSelector(&outil.MockSelector{
				SelFn: func(e *beat.Event) (string, error) {
					if e.Fields["outcome"] == "key-fail" {
						return "", errors.New("triggering key selection failure")
					}
					return "", nil
				},
			})
			c.codec = &encoderMock{encoder: json.New("", json.Config{})}

			conn := &redisMock{}
			tc.mockSetup(conn)
			c.publish = tc.makePublishFn(c, conn)

			for _, pc := range tc.publishCases {
				batch := outest.NewBatchWithObserver(listener, pc.events...)
				err = bc.Publish(context.Background(), batch)
				pc.assertFn(t, err)
			}

			outputtest.AssertOutputMetrics(t,
				tc.expectedMetrics,
				counter, reg)
		})
	}
}

type redisMock struct {
	receiveCount int
	receiveRet   []error

	doCounter int
	doRet     []error
}

type encoderMock struct {
	encoder codec.Codec
}

func (r *redisMock) Err() error {
	panic("implement me")
}

func (r *redisMock) Do(_ string, _ ...interface{}) (reply interface{}, err error) {
	idx := r.doCounter
	r.doCounter++
	return nil, r.doRet[idx]
}

func (r *redisMock) Send(_ string, _ ...interface{}) error {
	return nil
}

func (r *redisMock) Flush() error {
	return nil
}

func (r *redisMock) Receive() (_ interface{}, err error) {
	idx := r.receiveCount
	r.receiveCount++
	return nil, r.receiveRet[idx]
}

func (e encoderMock) Encode(index string, event *beat.Event) ([]byte, error) {
	if event.Fields["outcome"] == "encode-fail" {
		return nil, errors.New("triggering encoding failure")
	}

	return e.encoder.Encode(index, event)
}

func (r *redisMock) Close() error {
	return nil
}
