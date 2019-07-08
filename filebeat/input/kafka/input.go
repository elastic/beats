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
	goContext     context.Context
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

	//forwarder := harvester.NewForwarder(out)

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
	goContext, cancel := context.WithCancel(context.Background())
	go func() {
		select {
		case <-inputContext.Done:
			logp.Info("Closing kafka context because input stopped.")
			cancel()
			return
		}
	}()

	input := &Input{
		config:        config,
		rawConfig:     cfg,
		started:       false,
		outlet:        out,
		consumerGroup: consumerGroup,
		goContext:     goContext,
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
				// TODO: handle
				fmt.Println("ERROR", err)
			}
		}()

		go func() {
			for {
				handler := groupHandler{input: p}

				err := p.consumerGroup.Consume(p.goContext, p.config.Topics, handler)
				if err != nil {
					fmt.Printf("Consume error: %v\n", err)
					//panic(err)
					// TODO: report error
				}
			}
		}()
		p.started = true
	}
}

func (p *Input) Wait() {
}

func (p *Input) Stop() {
}

type groupHandler struct {
	input *Input
}

func createEvent(
	sess sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
	message *sarama.ConsumerMessage,
) *util.Data {
	data := util.NewData()
	data.Event = beat.Event{
		Timestamp: time.Now(),
		Fields: common.MapStr{
			"message": string(message.Value),
			"kafka": common.MapStr{
				"topic":     claim.Topic(),
				"partition": claim.Partition(),
			},
			// TODO: add more metadata
		},
	}
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
		event := createEvent(sess, claim, msg)
		fmt.Printf("event: %v\n", event)
		h.input.outlet.OnEvent(event)
		sess.MarkMessage(msg, "")
	}
	return nil
}
