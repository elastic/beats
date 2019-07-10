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

type initialOffset int

const (
	initialOffsetOldest initialOffset = iota
	initialOffsetNewest
)

var (
	defaultConfig = kafkaInputConfig{
		Version:       kafka.Version("1.0.0"),
		InitialOffset: initialOffsetOldest,
	}

	initialOffsets = map[string]initialOffset{
		"oldest": initialOffsetOldest,
		"newest": initialOffsetNewest,
	}
)

type kafkaInputConfig struct {
	// Kafka hosts with port, e.g. "localhost:9092"
	Hosts         []string      `config:"hosts" validate:"required"`
	Topics        []string      `config:"topics" validate:"required"`
	GroupID       string        `config:"group_id" validate:"required"`
	Version       kafka.Version `config:"version"`
	InitialOffset initialOffset `config:"initial_offset"`
}

func (off initialOffset) asSaramaOffset() int64 {
	return map[initialOffset]int64{
		initialOffsetOldest: sarama.OffsetOldest,
		initialOffsetNewest: sarama.OffsetNewest,
	}[off]
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
	k.Consumer.Offsets.Initial = config.InitialOffset.asSaramaOffset()

	if err := k.Validate(); err != nil {
		logp.Err("Invalid kafka configuration: %v", err)
		return nil, err
	}
	return k, nil
}

// Unpack validates and unpack the "initial_offset" config option
func (off *initialOffset) Unpack(value string) error {
	initialOffset, ok := initialOffsets[value]
	if !ok {
		return fmt.Errorf("invalid initialOffset '%s'", value)
	}

	*off = initialOffset

	return nil
}
