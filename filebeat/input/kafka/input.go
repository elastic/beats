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
	"strings"
	"sync"
	"time"

	"github.com/Shopify/sarama"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/backoff"
	"github.com/elastic/beats/libbeat/common/kafka"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/pkg/errors"
)

func init() {
	err := input.Register("kafka", NewInput)
	if err != nil {
		panic(err)
	}
}

// Input contains the input and its config
type kafkaInput struct {
	config          kafkaInputConfig
	saramaConfig    *sarama.Config
	context         input.Context
	outlet          channel.Outleter
	saramaWaitGroup sync.WaitGroup // indicates a sarama consumer group is active
	log             *logp.Logger
	runOnce         sync.Once
}

// NewInput creates a new kafka input
func NewInput(
	cfg *common.Config,
	connector channel.Connector,
	inputContext input.Context,
) (input.Input, error) {

	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, errors.Wrap(err, "reading kafka input config")
	}

	out, err := connector.ConnectWith(cfg, beat.ClientConfig{
		Processing: beat.ProcessingConfig{
			DynamicFields: inputContext.DynamicFields,
		},
		ACKEvents: func(events []interface{}) {
			for _, event := range events {
				if meta, ok := event.(eventMeta); ok {
					meta.handler.ack(meta.message)
				}
			}
		},
		CloseRef:  doneChannelContext(inputContext.Done),
		WaitClose: config.WaitClose,
	})
	if err != nil {
		return nil, err
	}

	saramaConfig, err := newSaramaConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "initializing Sarama config")
	}

	input := &kafkaInput{
		config:       config,
		saramaConfig: saramaConfig,
		context:      inputContext,
		outlet:       out,
		log:          logp.NewLogger("kafka input").With("hosts", config.Hosts),
	}

	return input, nil
}

func (input *kafkaInput) runConsumerGroup(
	context context.Context, consumerGroup sarama.ConsumerGroup,
) {
	handler := &groupHandler{
		version: input.config.Version,
		outlet:  input.outlet,
	}

	input.saramaWaitGroup.Add(1)
	defer func() {
		consumerGroup.Close()
		input.saramaWaitGroup.Done()
	}()

	// Listen asynchronously to any errors during the consume process
	go func() {
		for err := range consumerGroup.Errors() {
			input.log.Errorw("Error reading from kafka", "error", err)
		}
	}()

	err := consumerGroup.Consume(context, input.config.Topics, handler)
	if err != nil {
		input.log.Errorw("Kafka consume error", "error", err)
	}
}

// Run starts the input by scanning for incoming messages and errors.
func (input *kafkaInput) Run() {
	input.runOnce.Do(func() {
		go func() {
			// Sarama uses standard go contexts to control cancellation, so we need
			// to wrap our input context channel in that interface.
			context := doneChannelContext(input.context.Done)

			// If the consumer fails to connect, we use exponential backoff with
			// jitter up to 8 * the initial backoff interval.
			backoff := backoff.NewEqualJitterBackoff(
				input.context.Done,
				input.config.ConnectBackoff,
				8*input.config.ConnectBackoff)

			for context.Err() == nil {
				// Connect to Kafka with a new consumer group.
				consumerGroup, err := sarama.NewConsumerGroup(
					input.config.Hosts, input.config.GroupID, input.saramaConfig)
				if err != nil {
					input.log.Errorw(
						"Error initializing kafka consumer group", "error", err)
					backoff.Wait()
					continue
				}
				// We've successfully connected, reset the backoff timer.
				backoff.Reset()

				// We have a connected consumer group now, try to start the main event
				// loop by calling Consume (which starts an asynchronous consumer).
				// In an ideal run, this function never returns until shutdown; if it
				// does, it means the errors have been logged and the consumer group
				// has been closed, so we try creating a new one in the next iteration.
				input.runConsumerGroup(context, consumerGroup)
			}
		}()
	})
}

// Stop doesn't need to do anything because the kafka consumer group and the
// input's outlet both have a context based on input.context.Done and will
// shut themselves down, since the done channel is already closed as part of
// the shutdown process in Runner.Stop().
func (input *kafkaInput) Stop() {
}

// Wait should shut down the input and wait for it to complete, however (see
// Stop above) we don't need to take actions to shut down as long as the
// input.config.Done channel is closed, so we just make a (currently no-op)
// call to Stop() and then wait for sarama to signal completion.
func (input *kafkaInput) Wait() {
	input.Stop()
	// Wait for sarama to shut down
	input.saramaWaitGroup.Wait()
}

func arrayForKafkaHeaders(headers []*sarama.RecordHeader) []string {
	array := []string{}
	for _, header := range headers {
		// Rather than indexing headers in the same object structure Kafka does
		// (which would give maximal fidelity, but would be effectively unsearchable
		// in elasticsearch and kibana) we compromise by serializing them all as
		// strings in the form "<key>: <value>". For this we need to mask
		// occurrences of ":" in the original key, which we expect to be uncommon.
		// We may consider another approach in the future when it's more clear what
		// the most common use cases are.
		key := strings.ReplaceAll(string(header.Key), ":", "_")
		value := string(header.Value)
		array = append(array, fmt.Sprintf("%s: %s", key, value))
	}
	return array
}

// A barebones implementation of context.Context wrapped around the done
// channels that are more common in the beats codebase.
// TODO(faec): Generalize this to a common utility in a shared library
// (https://github.com/elastic/beats/issues/13125).
type channelCtx <-chan struct{}

func doneChannelContext(ch <-chan struct{}) context.Context {
	return channelCtx(ch)
}

func (c channelCtx) Deadline() (deadline time.Time, ok bool) { return }
func (c channelCtx) Done() <-chan struct{} {
	return (<-chan struct{})(c)
}
func (c channelCtx) Err() error {
	select {
	case <-c:
		return context.Canceled
	default:
		return nil
	}
}
func (c channelCtx) Value(key interface{}) interface{} { return nil }

// The group handler for the sarama consumer group interface. In addition to
// providing the basic consumption callbacks needed by sarama, groupHandler is
// also currently responsible for marshalling kafka messages into beat.Event,
// and passing ACKs from the output channel back to the kafka cluster.
type groupHandler struct {
	sync.Mutex
	version kafka.Version
	session sarama.ConsumerGroupSession
	outlet  channel.Outleter
}

// The metadata attached to incoming events so they can be ACKed once they've
// been successfully sent.
type eventMeta struct {
	handler *groupHandler
	message *sarama.ConsumerMessage
}

func (h *groupHandler) createEvent(
	sess sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
	message *sarama.ConsumerMessage,
) beat.Event {
	timestamp := time.Now()
	kafkaFields := common.MapStr{
		"topic":     claim.Topic(),
		"partition": claim.Partition(),
		"offset":    message.Offset,
		"key":       string(message.Key),
	}

	version, versionOk := h.version.Get()
	if versionOk && version.IsAtLeast(sarama.V0_10_0_0) {
		timestamp = message.Timestamp
		if !message.BlockTimestamp.IsZero() {
			kafkaFields["block_timestamp"] = message.BlockTimestamp
		}
	}
	if versionOk && version.IsAtLeast(sarama.V0_11_0_0) {
		kafkaFields["headers"] = arrayForKafkaHeaders(message.Headers)
	}
	event := beat.Event{
		Timestamp: timestamp,
		Fields: common.MapStr{
			"message": string(message.Value),
			"kafka":   kafkaFields,
		},
		Private: eventMeta{
			handler: h,
			message: message,
		},
	}

	return event
}

func (h *groupHandler) Setup(session sarama.ConsumerGroupSession) error {
	h.Lock()
	h.session = session
	h.Unlock()
	return nil
}

func (h *groupHandler) Cleanup(_ sarama.ConsumerGroupSession) error {
	h.Lock()
	h.session = nil
	h.Unlock()
	return nil
}

// ack informs the kafka cluster that this message has been consumed. Called
// from the input's ACKEvents handler.
func (h *groupHandler) ack(message *sarama.ConsumerMessage) {
	h.Lock()
	defer h.Unlock()
	if h.session != nil {
		h.session.MarkMessage(message, "")
	}
}

func (h *groupHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		event := h.createEvent(sess, claim, msg)
		h.outlet.OnEvent(event)
	}
	return nil
}
