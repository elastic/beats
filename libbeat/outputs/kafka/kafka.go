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
	"fmt"

	"github.com/Shopify/sarama"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/codec"
	"github.com/elastic/beats/v7/libbeat/outputs/outil"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	logSelector = "kafka"
)

func init() {
	sarama.Logger = kafkaLogger{log: logp.NewLogger(logSelector)}

	outputs.RegisterType("kafka", makeKafka)
}

func makeKafka(
	_ outputs.IndexManager,
	beat beat.Info,
	observer outputs.Observer,
	cfg *config.C,
) (outputs.Group, error) {
	log := logp.NewLogger(logSelector)
	log.Debug("initialize kafka output")

	kConfig, err := readConfig(cfg)
	if err != nil {
		return outputs.Fail(err)
	}

	topic, err := buildTopicSelector(cfg)
	if err != nil {
		return outputs.Fail(err)
	}

	libCfg, err := newSaramaConfig(log, kConfig)
	if err != nil {
		return outputs.Fail(err)
	}

	hosts, err := outputs.ReadHostList(cfg)
	if err != nil {
		return outputs.Fail(err)
	}

	codec, err := codec.CreateEncoder(beat, kConfig.Codec)
	if err != nil {
		return outputs.Fail(err)
	}

	client, err := newKafkaClient(observer, hosts, beat.IndexPrefix, kConfig.Key, topic, kConfig.Headers, codec, libCfg)
	if err != nil {
		return outputs.Fail(err)
	}

	retry := 0
	if kConfig.MaxRetries < 0 {
		retry = -1
	}
	return outputs.Success(kConfig.Queue, kConfig.BulkMaxSize, 0, retry, nil, client)
}

// buildTopicSelector builds the topic selector for standalone Beat and when
// running under Elastic-Agent based on cfg.
//
// When running standalone the topic selector works as expected and documented.
// When running under Elastic-Agent, dynamic topic selection is not supported,
// so a constant selector using the `topic` value is returned.
func buildTopicSelector(cfg *config.C) (outil.Selector, error) {
	topicCfg := struct {
		Topic string `config:"topic" yaml:"topic"`
	}{}

	if err := cfg.Unpack(&topicCfg); err != nil {
		return outil.Selector{}, fmt.Errorf("cannot unpack Kafka config to read the topic: %w", err)
	}

	if management.UnderAgent() {
		exprSelector := outil.ConstSelectorExpr(topicCfg.Topic, outil.SelectorKeepCase)
		selector := outil.MakeSelector(exprSelector)
		return selector, nil
	}

	return outil.BuildSelectorFromConfig(cfg, outil.Settings{
		Key:              "topic",
		MultiKey:         "topics",
		EnableSingleOnly: true,
		FailEmpty:        true,
		Case:             outil.SelectorKeepCase,
	})
}
