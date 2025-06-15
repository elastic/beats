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

package kafka

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/outputtest"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/pipeline"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/sarama"
	"github.com/elastic/sarama/mocks"
)

func TestClientOutputListener(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "",
		// only print stacktrace for errors above ErrorLevel.
		zap.AddStacktrace(zapcore.ErrorLevel+1))

	cfgSarama := sarama.NewConfig()
	cfgSarama.Producer.Return.Successes = true
	cfgSarama.Producer.Return.Errors = true

	producer := mocks.NewAsyncProducer(t, cfgSarama)
	// 1st event: succeed
	producer.ExpectInputAndSucceed()
	// 2nd event: permanent failure -> dropped
	producer.ExpectInputAndFail(
		fmt.Errorf("test permanent error: %w", sarama.ErrInvalidMessage))
	// 3rd event: retryable failure -> will trigger retryable error metrics
	producer.ExpectInputAndFail(
		fmt.Errorf("test retryable error: %w", sarama.ErrRequestTimedOut))

	cfg, err := config.NewConfigFrom(map[string]interface{}{
		"hosts":   []string{"localhost:9094"},
		"topic":   "testTopic",
		"timeout": "1s",
	})
	require.NoError(t, err, "could not create config from map")

	reg := monitoring.NewRegistry()
	outGrup, err := makeKafka(
		nil,
		beat.Info{
			Beat:        "libbeat",
			IndexPrefix: "testbeat",
			Logger:      logger},
		outputs.NewStats(reg), cfg)
	require.NoError(t, err, "could not create kafka output")

	c, ok := outGrup.Clients[0].(*client)
	require.Truef(t, ok, "Expected output to be of type %T", &client{})

	c.producer = producer
	c.wg.Add(2)
	go c.successWorker(c.producer.Successes())
	go c.errorWorker(c.producer.Errors())

	counter := &beat.CountOutputListener{}
	observer := publisher.OutputListener{Listener: counter}
	b := pipeline.MockBatch{
		Mu: sync.Mutex{},
		EventList: []publisher.Event{
			{
				OutputListener: observer,
				Content: beat.Event{
					Timestamp: time.Time{},
					Meta:      nil,
					Fields: map[string]interface{}{
						"msg":     "message 1",
						"to_drop": "false"},
					Private:    nil,
					TimeSeries: false,
				},
			},
			{
				OutputListener: observer,
				Content: beat.Event{
					Timestamp: time.Time{},
					Meta:      nil,
					Fields: map[string]interface{}{
						"msg":     "message 2",
						"to_drop": "true"},
					Private:    nil,
					TimeSeries: false,
				},
			},
			{
				OutputListener: observer,
				Content: beat.Event{
					Timestamp: time.Time{},
					Meta:      nil,
					Fields: map[string]interface{}{
						"msg":      "message 3",
						"to_retry": "true"},
					Private:    nil,
					TimeSeries: false,
				},
			},
		},
	}

	err = c.Publish(context.Background(), &b)
	require.NoError(t, err, "could not publish batch")

	require.NoError(t, c.Close(), "failed closing kafka client")

	outputtest.AssertOutputMetrics(t,
		outputtest.Metrics{
			Total:     3,
			Acked:     1,
			Dropped:   1,
			Retryable: 1,
			Batches:   1,
		},
		counter, reg)
}
