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
	"sync"
	"time"

	"github.com/Shopify/sarama"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
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
	config        kafkaInputConfig
	rawConfig     *common.Config // The Config given to NewInput
	outlet        channel.Outleter
	consumerGroup sarama.ConsumerGroup
	sessionState  *kafkaSessionState
	kafkaContext  context.Context
	kafkaCancel   context.CancelFunc // The CancelFunc for kafkaContext
	log           *logp.Logger
	runOnce       sync.Once
}

// A synchronized wrapper to read and write the kafka session, since it may
// change while ACKs are still pending.
type kafkaSessionState struct {
	session   sarama.ConsumerGroupSession
	mutex     sync.Mutex     // Hold to access the session field
	waitGroup sync.WaitGroup // Hold while using the session field
}

// NewInput creates a new kafka input
func NewInput(
	cfg *common.Config,
	connector channel.Connector,
	inputContext input.Context,
) (input.Input, error) {

	// We create the empty session state first because it must be referenced by
	// the ACK callback in the connector configuration.
	sessionState := &kafkaSessionState{}

	out, err := connector.ConnectWith(cfg, beat.ClientConfig{
		Processing: beat.ProcessingConfig{
			DynamicFields: inputContext.DynamicFields,
		},
		ACKEvents: func(events []interface{}) {
			sessionState.accessSession(func(session sarama.ConsumerGroupSession) {
				if session == nil {
					// The kafka connection is closed and / or is being rebalanced.
					return
				}
				for _, event := range events {
					if cm, ok := event.(*sarama.ConsumerMessage); ok {
						session.MarkMessage(cm, "")
					}
				}
			})
		},
	})
	if err != nil {
		return nil, err
	}

	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, errors.Wrap(err, "reading kafka input config")
	}

	saramaConfig, err := newSaramaConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "initializing Sarama config")
	}
	consumerGroup, err :=
		sarama.NewConsumerGroup(config.Hosts, config.GroupID, saramaConfig)
	if err != nil {
		return nil, errors.Wrap(err, "initializing kafka consumer group")
	}

	// Sarama uses standard go contexts to control cancellation, so we need to
	// wrap our input context channel in that interface.
	kafkaContext, kafkaCancel := context.WithCancel(context.Background())
	go func() {
		select {
		case <-inputContext.Done:
			logp.Info("Closing kafka context because input stopped.")
			kafkaCancel()
			return
		}
	}()

	input := &kafkaInput{
		config:        config,
		rawConfig:     cfg,
		outlet:        out,
		consumerGroup: consumerGroup,
		sessionState:  sessionState,
		kafkaContext:  kafkaContext,
		kafkaCancel:   kafkaCancel,
		log:           logp.NewLogger("kafka input").With("hosts", config.Hosts),
	}

	return input, nil
}

// A helper to safely use the current sarama session for the duration of the
// given callback. Used when ACKing messages outside the body of the main
// sarama callbacks. The session parameter may be nil if there is no active
// session.
func (state *kafkaSessionState) accessSession(
	fn func(session sarama.ConsumerGroupSession),
) {
	state.mutex.Lock()
	state.waitGroup.Add(1)
	session := state.session
	state.mutex.Unlock()
	defer state.waitGroup.Done()
	fn(session)
}

// A helper to safely set the session field after waiting on any pending
// operations.
func (state *kafkaSessionState) setSession(sess sarama.ConsumerGroupSession) {
	state.mutex.Lock()
	// Once we claim the mutex we still wait for any pending ACKs to be
	// sent. (These may well fail if the session is ending, but that's better
	// than calling a stale pointer.)
	state.waitGroup.Wait()
	state.session = sess
	state.mutex.Unlock()
}

// Run starts the input by scanning for incoming messages and errors.
func (input *kafkaInput) Run() {
	input.runOnce.Do(func() {
		// Track errors
		go func() {
			for err := range input.consumerGroup.Errors() {
				input.log.Errorw("Error reading from kafka", "error", err)
			}
		}()

		go func() {
			for {
				handler := groupHandler{input: input}

				err := input.consumerGroup.Consume(
					input.kafkaContext, input.config.Topics, handler)
				if err != nil {
					input.log.Errorw("Kafka consume error", "error", err)
				}
			}
		}()
	})
}

// Wait shuts down the Input by cancelling the internal context.
func (input *kafkaInput) Wait() {
	input.Stop()
	// Wait for the consumer group to shut down
	input.consumerGroup.Close()
}

// Stop shuts down the Input by cancelling the internal context.
func (input *kafkaInput) Stop() {
	input.kafkaCancel()
}

func arrayForKafkaHeaders(headers []*sarama.RecordHeader) []interface{} {
	array := []interface{}{}
	for _, header := range headers {
		array = append(array, common.MapStr{
			"key":   header.Key,
			"value": header.Value,
		})
	}
	return array
}

type groupHandler struct {
	input *kafkaInput
}

func (h groupHandler) createEvent(
	sess sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
	message *sarama.ConsumerMessage,
) beat.Event {
	event := beat.Event{
		Timestamp: time.Now(),
	}
	eventFields := common.MapStr{
		"message": string(message.Value),
	}
	kafkaMetadata := common.MapStr{
		"topic":     claim.Topic(),
		"partition": claim.Partition(),
		"offset":    message.Offset,
		"key":       message.Key,
	}
	version, versionOk := h.input.config.Version.Get()
	if versionOk && version.IsAtLeast(sarama.V0_10_0_0) {
		event.Timestamp = message.Timestamp
		if !message.BlockTimestamp.IsZero() {
			kafkaMetadata["block_timestamp"] = message.BlockTimestamp
		}
	}
	if versionOk && version.IsAtLeast(sarama.V0_11_0_0) {
		kafkaMetadata["headers"] = arrayForKafkaHeaders(message.Headers)
	}
	eventFields["kafka"] = kafkaMetadata
	event.Fields = eventFields
	return event
}

func (h groupHandler) Setup(session sarama.ConsumerGroupSession) error {
	h.input.sessionState.setSession(session)
	return nil
}

func (h groupHandler) Cleanup(_ sarama.ConsumerGroupSession) error {
	h.input.sessionState.setSession(nil)
	return nil
}

func (h groupHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		event := h.createEvent(sess, claim, msg)
		h.input.outlet.OnEvent(event)
		sess.MarkMessage(msg, "")
	}
	return nil
}
