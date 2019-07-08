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
	"github.com/elastic/beats/libbeat/common/kafka"
	"github.com/elastic/beats/libbeat/logp"
)

var defaultConfig = kafkaInputConfig{
	Version: kafka.Version("1.0.0"),
	GroupID: "FilebeatGroup",
}

type kafkaInputConfig struct {
	// Kafka hosts with port, e.g. "localhost:9092"
	Hosts   []string      `config:"hosts" validate:"required"`
	Topics  []string      `config:"topics" validate:"required"`
	Version kafka.Version `config:"version"`
	GroupID string        `config:"group_id"`
}

// Validate validates the config.
func (c *kafkaInputConfig) Validate() error {

	return nil
}

func stringInSlice(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}

func newSaramaConfig(config kafkaInputConfig) (*sarama.Config, error) {
	k := sarama.NewConfig()

	version, ok := config.Version.Get()
	if !ok {
		return nil, fmt.Errorf("Unknown/unsupported kafka version: %v", config.Version)
	}
	k.Version = version

	k.Consumer.Return.Errors = true
	k.Consumer.Offsets.Initial = sarama.OffsetOldest

	if err := k.Validate(); err != nil {
		logp.Err("Invalid kafka configuration: %v", err)
		return nil, err
	}
	return k, nil
}
