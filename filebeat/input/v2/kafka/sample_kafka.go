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

//+build donotbuild
package kafka

// NOTE: This file pseudo-implements the kafka input using the v2 plugin interface.
//       It doesn't compile, but is used to get an idea on how the v2 plugin interface
//       will be used by input implementors.

import (
	"sync"

	"github.com/Shopify/sarama"
	"github.com/pkg/errors"

	input "github.com/elastic/beats/filebeat/input/v2"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/backoff"
	"github.com/elastic/beats/libbeat/common/kafka"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/go-concert/chorus"
)

var Plugin = input.Plugin{
	Name:      "kafka",
	Doc:       "Collect kafka topics using consumer groups",
	Configure: initInput,
}

type kafkaInput struct {
	config       inputConfig
	saramaConfig *sarama.Config
}

type inputConfig struct {
	// ...
}

type groupHandler struct {
	sync.Mutex
	log                      *logp.Logger
	version                  kafka.Version
	session                  sarama.ConsumerGroupSession
	out                      beat.Client
	expandEventListFromField string
}

type eventMeta struct {
	handler *groupHandler
	message *sarama.ConsumerMessage
}

func initInput(log *logp.Logger, cfg *common.Config) (input.Input, error) {
	config, err := parseConfig(cfg)
	if err != nil {
		return nil, err
	}

	saramaConfig, err := newSaramaConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "initializing sarama config")
	}

	return input.ConfiguredInput{
		Info: "kafka",
		Input: &kafkaInput{
			config:       config,
			saramaConfig: saramaConfig,
		}
	}, nil
}

func parseConfig(config *common.Config) (inputConfig, error) {
	c := defaultConfig()
	err := cfg.Unpack(&config)
	if err != nil {
		err = errors.Wrap(err, "reading kafka input config")
	}

	return c, err
}

func defaultConfig() inputConfig {
	return inputConfig{ /* ... */ }
}

func newSaramaConfig(config kafkaInputConfig) (*sarama.Config, error) {
	/* ... */
	return nil, nil
}

func (inst *kafkaInput) TestInput(closer *chorus.Closer, log *logp.Logger) error {
	// TODO: try to connect, check if topics are available
	return nil
}

func (inst *kafkaInput) Run(context input.Context) error {
	log := context.Log // <- TODO: add context like consumer group name and endpoints

	// NOTE: the runner keeps track of pipeline connections and auto-closes them
	//       if Run returns. Closing a connection within Run is optional.
	//       The input runner automatically adds context.Closer, if no custom CloseRef
	//       is configured.
	out := context.Pipeline.ConnectWith(beat.ClientConfig{
		ACKEvents: func(events []interface{}) {
			for _, event := range events {
				if meta, ok := event.(eventMeta); ok {
					meta.handler.ack(meta.message)
				}
			}
		},
		WaitClose: inst.config.WaitClose,
	})

	// If the consumer fails to connect, we use exponential backoff with
	// jitter up to 8 * the initial backoff interval.
	backoff := backoff.NewEqualJitterBackoff(
		inst.context.Done,
		inst.config.ConnectBackoff,
		8*inst.config.ConnectBackoff)

	inst.context.Status.Initialized()

	for context.Closer.Err() == nil { // restart input after error (given there is no shutdown signal)
		group, err := sarama.NewConsumerGroup(inst.config.Hosts, inst.config.GroupID, inst.saramaConfig)
		if err != nil {
			if err == chorus.ErrClosed {
				break
			}

			inst.context.Status.Failing(err)
			log.Errorw(
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
		inst.context.Status.Active()
		err := inst.runConsumerGroup(context, out, consumerGroup)
		if err != nil && err != chorus.ErrClosed {
			inst.context.Status.Failing(err)
		}
	}
	inst.context.Status.Stopping()

	return nil
}

func (inst *kafkaInput) runConsumerGroup(
	log *logp.Logger,
	context input.Context,
	out beat.Client,
	group sarama.ConsumerGroup,
) error {
	defer group.Close()

	handler := &groupHandler{
		version: inst.config.Version,
		out:     out,
		// expandEventListFromField will be assigned the configuration option expand_event_list_from_field
		expandEventListFromField: inst.config.ExpandEventListFromField,
		log:                      log,
	}

	// Listen asynchronously to any errors during the consume process
	go func() {
		for err := range consumerGroup.Errors() {
			log.Errorw("Error reading from kafka", "error", err)
		}
	}()

	err := consumerGroup.Consume(context.Closer, input.config.Topics, handler)
	if err != nil {
		log.Errorw("Kafka consume error", "error", err)
	}
	return err
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

func (h *groupHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		h.out.PublishAll(h.createEvents(sess, claim, msg))
	}
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

func (h *groupHandler) createEvents(
	sess sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
	message *sarama.ConsumerMessage,
) []beat.Event {
	/* ... */
	return nil
}
