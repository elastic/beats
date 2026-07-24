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

//go:build !integration

package kafka

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/resources"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/sarama"
)

func TestNewInputDone(t *testing.T) {
	config := conf.MustNewConfigFrom(mapstr.M{
		"hosts":    "localhost:9092",
		"topics":   "messages",
		"group_id": "filebeat",
	})

	AssertNotStartedInputCanBeDone(t, config)
}

// AssertNotStartedInputCanBeDone checks that the context of an input can be
// done before starting the input, and it doesn't leak goroutines. This is
// important to confirm that leaks don't happen with CheckConfig.
func AssertNotStartedInputCanBeDone(t *testing.T, configMap *conf.C) {
	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.Check(t)

	config, err := conf.NewConfigFrom(configMap)
	require.NoError(t, err)

	_, err = Plugin(logp.NewNopLogger()).Manager.Create(config)
	require.NoError(t, err)
}

func TestDeadlineReceiver(t *testing.T) {
	t.Run("no deadline blocks until a message arrives", func(t *testing.T) {
		ch := make(chan *sarama.ConsumerMessage, 1)
		ch <- &sarama.ConsumerMessage{Value: []byte("x")}
		var d deadlineReceiver
		msg, ok, timedOut := d.recv(ch)
		require.True(t, ok)
		require.False(t, timedOut)
		require.Equal(t, []byte("x"), msg.Value)
	})

	t.Run("fast path returns a buffered message without arming the timer", func(t *testing.T) {
		ch := make(chan *sarama.ConsumerMessage, 1)
		ch <- &sarama.ConsumerMessage{Value: []byte("y")}
		var d deadlineReceiver
		d.SetReadDeadline(time.Now().Add(time.Hour))
		msg, ok, timedOut := d.recv(ch)
		require.True(t, ok)
		require.False(t, timedOut)
		require.Equal(t, []byte("y"), msg.Value)
		require.Nil(t, d.timer, "fast path must not allocate a timer")
	})

	t.Run("times out on an empty channel", func(t *testing.T) {
		ch := make(chan *sarama.ConsumerMessage)
		var d deadlineReceiver
		d.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
		_, _, timedOut := d.recv(ch)
		require.True(t, timedOut)
	})

	t.Run("reuses a single timer across slow-path calls", func(t *testing.T) {
		ch := make(chan *sarama.ConsumerMessage)
		var d deadlineReceiver
		d.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
		_, _, timedOut := d.recv(ch)
		require.True(t, timedOut)
		first := d.timer
		require.NotNil(t, first)

		d.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
		_, _, timedOut = d.recv(ch)
		require.True(t, timedOut)
		require.Same(t, first, d.timer, "the timer must be reused, not reallocated")
	})
}
