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
	"time"

	"github.com/Shopify/sarama"
	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/filebeat/util"
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
type Input struct {
	config        kafkaInputConfig
	rawConfig     *common.Config // The Config given to NewInput
	started       bool
	outlet        channel.Outleter
	consumerGroup sarama.ConsumerGroup
	kafkaContext  context.Context
	kafkaCancel   context.CancelFunc // The CancelFunc for kafkaContext
	log           *logp.Logger
}

// NewInput creates a new kafka input
func NewInput(
	cfg *common.Config,
	outletFactory channel.Connector,
	inputContext input.Context,
) (input.Input, error) {

	out, err := outletFactory(cfg, inputContext.DynamicFields)
	if err != nil {
		return nil, err
	}

	config := defaultConfig
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

	input := &Input{
		config:        config,
		rawConfig:     cfg,
		started:       false,
		outlet:        out,
		consumerGroup: consumerGroup,
		kafkaContext:  kafkaContext,
		kafkaCancel:   kafkaCancel,
		log:           logp.NewLogger("kafka input").With("hosts", config.Hosts),
	}

	return input, nil
}

func (p *Input) newConsumerGroup() (sarama.ConsumerGroup, error) {
	consumerGroup, err :=
		sarama.NewConsumerGroup(p.config.Hosts, p.config.GroupID, nil)
	return consumerGroup, err
}

// Run starts the input by scanning for incoming messages and errors.
func (p *Input) Run() {
	if !p.started {
		// Track errors
		go func() {
			for err := range p.consumerGroup.Errors() {
				p.log.Errorw("Error reading from kafka", "error", err)
			}
		}()

		go func() {
			for {
				handler := groupHandler{input: p}

				err := p.consumerGroup.Consume(p.kafkaContext, p.config.Topics, handler)
				if err != nil {
					p.log.Errorw("Kafka consume error", "error", err)
				}
			}
		}()
		p.started = true
	}
}

// Wait shuts down the Input by cancelling the internal context.
func (p *Input) Wait() {
	p.Stop()
}

// Stop shuts down the Input by cancelling the internal context.
func (p *Input) Stop() {
	p.kafkaCancel()
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
	input *Input
}

func (h groupHandler) createEvent(
	sess sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
	message *sarama.ConsumerMessage,
) *util.Data {
	data := util.NewData()
	data.Event = beat.Event{
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
		data.Event.Timestamp = message.Timestamp
		if !message.BlockTimestamp.IsZero() {
			kafkaMetadata["block_timestamp"] = message.BlockTimestamp
		}
	}
	if versionOk && version.IsAtLeast(sarama.V0_11_0_0) {
		kafkaMetadata["headers"] = arrayForKafkaHeaders(message.Headers)
	}
	eventFields["kafka"] = kafkaMetadata
	data.Event.Fields = eventFields
	return data
}

func (groupHandler) Setup(session sarama.ConsumerGroupSession) error {
	return nil
}

func (groupHandler) Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}

func (h groupHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		event := h.createEvent(sess, claim, msg)
		fmt.Printf("event: %v\n", event)
		h.input.outlet.OnEvent(event)
		sess.MarkMessage(msg, "")
	}
	return nil
}
