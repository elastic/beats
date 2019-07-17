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
	"errors"
	"fmt"

	"github.com/Shopify/sarama"

	"github.com/elastic/beats/libbeat/common/kafka"
	"github.com/elastic/beats/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
	"github.com/elastic/beats/libbeat/monitoring/adapter"
	"github.com/elastic/beats/libbeat/outputs"
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
		ClientID:      "filebeat",
	}

	initialOffsets = map[string]initialOffset{
		"oldest": initialOffsetOldest,
		"newest": initialOffsetNewest,
	}
)

type kafkaInputConfig struct {
	// Kafka hosts with port, e.g. "localhost:9092"
	Hosts         []string          `config:"hosts" validate:"required"`
	Topics        []string          `config:"topics" validate:"required"`
	GroupID       string            `config:"group_id" validate:"required"`
	ClientID      string            `config:"client_id"`
	Version       kafka.Version     `config:"version"`
	InitialOffset initialOffset     `config:"initial_offset"`
	TLS           *tlscommon.Config `config:"ssl"`
	Username      string            `config:"username"`
	Password      string            `config:"password"`
}

// Validate validates the config.
func (c *kafkaInputConfig) Validate() error {
	if len(c.Hosts) == 0 {
		return errors.New("no hosts configured")
	}

	if err := c.Version.Validate(); err != nil {
		return err
	}

	if c.Username != "" && c.Password == "" {
		return fmt.Errorf("password must be set when username is configured")
	}
	return nil
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

	tls, err := outputs.LoadTLSConfig(config.TLS)
	if err != nil {
		return nil, err
	}
	if tls != nil {
		k.Net.TLS.Enable = true
		k.Net.TLS.Config = tls.BuildModuleConfig("")
	}

	if config.Username != "" {
		k.Net.SASL.Enable = true
		k.Net.SASL.User = config.Username
		k.Net.SASL.Password = config.Password
	}

	// configure client ID
	k.ClientID = config.ClientID

	k.MetricRegistry = adapter.GetGoMetrics(
		monitoring.Default,
		"filebeat.inputs.kafka",
		adapter.Rename("incoming-byte-rate", "bytes_read"),
		adapter.Rename("outgoing-byte-rate", "bytes_write"),
		adapter.GoMetricsNilify,
	)

	if err := k.Validate(); err != nil {
		logp.Err("Invalid kafka configuration: %v", err)
		return nil, err
	}
	return k, nil
}

// asSaramaOffset converts an initialOffset enum to the corresponding
// sarama offset value.
func (off initialOffset) asSaramaOffset() int64 {
	return map[initialOffset]int64{
		initialOffsetOldest: sarama.OffsetOldest,
		initialOffsetNewest: sarama.OffsetNewest,
	}[off]
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
