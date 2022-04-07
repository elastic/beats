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
	"time"

	"github.com/Shopify/sarama"

	"github.com/elastic/beats/v8/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v8/libbeat/common/kafka"
	"github.com/elastic/beats/v8/libbeat/common/transport/kerberos"
	"github.com/elastic/beats/v8/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v8/libbeat/monitoring"
	"github.com/elastic/beats/v8/libbeat/monitoring/adapter"
	"github.com/elastic/beats/v8/libbeat/reader/parser"
)

type kafkaInputConfig struct {
	// Kafka hosts with port, e.g. "localhost:9092"
	Hosts                    []string          `config:"hosts" validate:"required"`
	Topics                   []string          `config:"topics" validate:"required"`
	GroupID                  string            `config:"group_id" validate:"required"`
	ClientID                 string            `config:"client_id"`
	Version                  kafka.Version     `config:"version"`
	InitialOffset            initialOffset     `config:"initial_offset"`
	ConnectBackoff           time.Duration     `config:"connect_backoff" validate:"min=0"`
	ConsumeBackoff           time.Duration     `config:"consume_backoff" validate:"min=0"`
	WaitClose                time.Duration     `config:"wait_close" validate:"min=0"`
	MaxWaitTime              time.Duration     `config:"max_wait_time"`
	IsolationLevel           isolationLevel    `config:"isolation_level"`
	Fetch                    kafkaFetch        `config:"fetch"`
	Rebalance                kafkaRebalance    `config:"rebalance"`
	TLS                      *tlscommon.Config `config:"ssl"`
	Kerberos                 *kerberos.Config  `config:"kerberos"`
	Username                 string            `config:"username"`
	Password                 string            `config:"password"`
	Sasl                     kafka.SaslConfig  `config:"sasl"`
	ExpandEventListFromField string            `config:"expand_event_list_from_field"`
	Parsers                  parser.Config     `config:",inline"`
}

type kafkaFetch struct {
	Min     int32 `config:"min" validate:"min=1"`
	Default int32 `config:"default" validate:"min=1"`
	Max     int32 `config:"max" validate:"min=0"`
}

type kafkaRebalance struct {
	Strategy     rebalanceStrategy `config:"strategy"`
	Timeout      time.Duration     `config:"timeout"`
	MaxRetries   int               `config:"max_retries"`
	RetryBackoff time.Duration     `config:"retry_backoff" validate:"min=0"`
}

type initialOffset int

const (
	initialOffsetOldest initialOffset = iota
	initialOffsetNewest
)

type rebalanceStrategy int

const (
	rebalanceStrategyRange rebalanceStrategy = iota
	rebalanceStrategyRoundRobin
)

type isolationLevel int

const (
	isolationLevelReadUncommitted = iota
	isolationLevelReadCommitted
)

var (
	initialOffsets = map[string]initialOffset{
		"oldest": initialOffsetOldest,
		"newest": initialOffsetNewest,
	}
	rebalanceStrategies = map[string]rebalanceStrategy{
		"range":      rebalanceStrategyRange,
		"roundrobin": rebalanceStrategyRoundRobin,
	}
	isolationLevels = map[string]isolationLevel{
		"read_uncommitted": isolationLevelReadUncommitted,
		"read_committed":   isolationLevelReadCommitted,
	}
)

// The default config for the kafka input. When in doubt, default values
// were chosen to match sarama's defaults.
func defaultConfig() kafkaInputConfig {
	return kafkaInputConfig{
		Version:        kafka.Version("1.0.0"),
		InitialOffset:  initialOffsetOldest,
		ClientID:       "filebeat",
		ConnectBackoff: 30 * time.Second,
		ConsumeBackoff: 2 * time.Second,
		WaitClose:      2 * time.Second,
		MaxWaitTime:    250 * time.Millisecond,
		IsolationLevel: isolationLevelReadUncommitted,
		Fetch: kafkaFetch{
			Min:     1,
			Default: (1 << 20), // 1 MB
			Max:     0,
		},
		Rebalance: kafkaRebalance{
			Strategy:     rebalanceStrategyRange,
			Timeout:      60 * time.Second,
			MaxRetries:   4,
			RetryBackoff: 2 * time.Second,
		},
	}
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
		return nil, fmt.Errorf("unknown/unsupported kafka version: %v", config.Version)
	}
	k.Version = version

	k.Consumer.Return.Errors = true
	k.Consumer.Offsets.Initial = config.InitialOffset.asSaramaOffset()
	k.Consumer.Retry.Backoff = config.ConsumeBackoff
	k.Consumer.MaxWaitTime = config.MaxWaitTime
	k.Consumer.IsolationLevel = config.IsolationLevel.asSaramaIsolationLevel()

	k.Consumer.Fetch.Min = config.Fetch.Min
	k.Consumer.Fetch.Default = config.Fetch.Default
	k.Consumer.Fetch.Max = config.Fetch.Max

	k.Consumer.Group.Rebalance.Strategy = config.Rebalance.Strategy.asSaramaStrategy()
	k.Consumer.Group.Rebalance.Timeout = config.Rebalance.Timeout
	k.Consumer.Group.Rebalance.Retry.Backoff = config.Rebalance.RetryBackoff
	k.Consumer.Group.Rebalance.Retry.Max = config.Rebalance.MaxRetries

	tls, err := tlscommon.LoadTLSConfig(config.TLS)
	if err != nil {
		return nil, err
	}
	if tls != nil {
		k.Net.TLS.Enable = true
		k.Net.TLS.Config = tls.BuildModuleClientConfig("")
	}

	if config.Kerberos.IsEnabled() {
		cfgwarn.Beta("Kerberos authentication for Kafka is beta.")

		k.Net.SASL.Enable = true
		k.Net.SASL.Mechanism = sarama.SASLTypeGSSAPI
		k.Net.SASL.GSSAPI = sarama.GSSAPIConfig{
			AuthType:           int(config.Kerberos.AuthType),
			KeyTabPath:         config.Kerberos.KeyTabPath,
			KerberosConfigPath: config.Kerberos.ConfigPath,
			ServiceName:        config.Kerberos.ServiceName,
			Username:           config.Kerberos.Username,
			Password:           config.Kerberos.Password,
			Realm:              config.Kerberos.Realm,
			DisablePAFXFAST:    !config.Kerberos.EnableFAST,
		}
	} else if config.Username != "" {
		k.Net.SASL.Enable = true
		k.Net.SASL.User = config.Username
		k.Net.SASL.Password = config.Password
		config.Sasl.ConfigureSarama(k)
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

func (st rebalanceStrategy) asSaramaStrategy() sarama.BalanceStrategy {
	return map[rebalanceStrategy]sarama.BalanceStrategy{
		rebalanceStrategyRange:      sarama.BalanceStrategyRange,
		rebalanceStrategyRoundRobin: sarama.BalanceStrategyRoundRobin,
	}[st]
}

// Unpack validates and unpack the "rebalance.strategy" config option
func (st *rebalanceStrategy) Unpack(value string) error {
	strategy, ok := rebalanceStrategies[value]
	if !ok {
		return fmt.Errorf("invalid rebalance strategy '%s'", value)
	}
	*st = strategy
	return nil
}

func (is isolationLevel) asSaramaIsolationLevel() sarama.IsolationLevel {
	return map[isolationLevel]sarama.IsolationLevel{
		isolationLevelReadUncommitted: sarama.ReadUncommitted,
		isolationLevelReadCommitted:   sarama.ReadCommitted,
	}[is]
}

func (is *isolationLevel) Unpack(value string) error {
	isolationLevel, ok := isolationLevels[value]
	if !ok {
		return fmt.Errorf("invalid isolation level '%s'", value)
	}
	*is = isolationLevel
	return nil
}
