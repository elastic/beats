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
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/Shopify/sarama"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/common/fmtstr"
	"github.com/elastic/beats/v7/libbeat/common/kafka"
	"github.com/elastic/beats/v7/libbeat/common/transport/kerberos"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/monitoring"
	"github.com/elastic/beats/v7/libbeat/monitoring/adapter"
	"github.com/elastic/beats/v7/libbeat/outputs/codec"
)

type backoffConfig struct {
	Init time.Duration `config:"init"`
	Max  time.Duration `config:"max"`
}

type kafkaConfig struct {
	Hosts              []string                  `config:"hosts"               validate:"required"`
	TLS                *tlscommon.Config         `config:"ssl"`
	Kerberos           *kerberos.Config          `config:"kerberos"`
	Timeout            time.Duration             `config:"timeout"             validate:"min=1"`
	Metadata           metaConfig                `config:"metadata"`
	Key                *fmtstr.EventFormatString `config:"key"`
	Partition          map[string]*common.Config `config:"partition"`
	KeepAlive          time.Duration             `config:"keep_alive"          validate:"min=0"`
	MaxMessageBytes    *int                      `config:"max_message_bytes"   validate:"min=1"`
	RequiredACKs       *int                      `config:"required_acks"       validate:"min=-1"`
	BrokerTimeout      time.Duration             `config:"broker_timeout"      validate:"min=1"`
	Compression        string                    `config:"compression"`
	CompressionLevel   int                       `config:"compression_level"`
	Version            kafka.Version             `config:"version"`
	BulkMaxSize        int                       `config:"bulk_max_size"`
	BulkFlushFrequency time.Duration             `config:"bulk_flush_frequency"`
	MaxRetries         int                       `config:"max_retries"         validate:"min=-1,nonzero"`
	Backoff            backoffConfig             `config:"backoff"`
	ClientID           string                    `config:"client_id"`
	ChanBufferSize     int                       `config:"channel_buffer_size" validate:"min=1"`
	Username           string                    `config:"username"`
	Password           string                    `config:"password"`
	Codec              codec.Config              `config:"codec"`
	Sasl               saslConfig                `config:"sasl"`
}

type saslConfig struct {
	SaslMechanism string `config:"mechanism"`
	//SaslUsername  string `config:"username"` //maybe use ssl.username ssl.password instead in future?
	//SaslPassword  string `config:"password"`
}

type metaConfig struct {
	Retry       metaRetryConfig `config:"retry"`
	RefreshFreq time.Duration   `config:"refresh_frequency" validate:"min=0"`
	Full        bool            `config:"full"`
}

type metaRetryConfig struct {
	Max     int           `config:"max"     validate:"min=0"`
	Backoff time.Duration `config:"backoff" validate:"min=0"`
}

var compressionModes = map[string]sarama.CompressionCodec{
	// As of sarama 1.24.1, zstd support is broken
	// (https://github.com/Shopify/sarama/issues/1252), which needs to be
	// addressed before we add support here.
	"none":   sarama.CompressionNone,
	"no":     sarama.CompressionNone,
	"off":    sarama.CompressionNone,
	"gzip":   sarama.CompressionGZIP,
	"lz4":    sarama.CompressionLZ4,
	"snappy": sarama.CompressionSnappy,
}

const (
	saslTypePlaintext   = sarama.SASLTypePlaintext
	saslTypeSCRAMSHA256 = sarama.SASLTypeSCRAMSHA256
	saslTypeSCRAMSHA512 = sarama.SASLTypeSCRAMSHA512
)

func defaultConfig() kafkaConfig {
	return kafkaConfig{
		Hosts:              nil,
		TLS:                nil,
		Kerberos:           nil,
		Timeout:            30 * time.Second,
		BulkMaxSize:        2048,
		BulkFlushFrequency: 0,
		Metadata: metaConfig{
			Retry: metaRetryConfig{
				Max:     3,
				Backoff: 250 * time.Millisecond,
			},
			RefreshFreq: 10 * time.Minute,
			Full:        false,
		},
		KeepAlive:        0,
		MaxMessageBytes:  nil, // use library default
		RequiredACKs:     nil, // use library default
		BrokerTimeout:    10 * time.Second,
		Compression:      "gzip",
		CompressionLevel: 4,
		Version:          kafka.Version("1.0.0"),
		MaxRetries:       3,
		Backoff: backoffConfig{
			Init: 1 * time.Second,
			Max:  60 * time.Second,
		},
		ClientID:       "beats",
		ChanBufferSize: 256,
		Username:       "",
		Password:       "",
	}
}

func (c *saslConfig) configureSarama(config *sarama.Config) error {
	switch strings.ToUpper(c.SaslMechanism) { // try not to force users to use all upper case
	case "":
		// SASL is not enabled
		return nil
	case saslTypePlaintext:
		config.Net.SASL.Mechanism = sarama.SASLMechanism(sarama.SASLTypePlaintext)
	case saslTypeSCRAMSHA256:
		config.Net.SASL.Handshake = true
		config.Net.SASL.Mechanism = sarama.SASLMechanism(sarama.SASLTypeSCRAMSHA256)
		config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
			return &XDGSCRAMClient{HashGeneratorFcn: SHA256}
		}
	case saslTypeSCRAMSHA512:
		config.Net.SASL.Handshake = true
		config.Net.SASL.Mechanism = sarama.SASLMechanism(sarama.SASLTypeSCRAMSHA512)
		config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
			return &XDGSCRAMClient{HashGeneratorFcn: SHA512}
		}
	default:
		return fmt.Errorf("not valid mechanism '%v', only supported with PLAIN|SCRAM-SHA-512|SCRAM-SHA-256", c.SaslMechanism)
	}

	return nil
}

func readConfig(cfg *common.Config) (*kafkaConfig, error) {
	c := defaultConfig()
	if err := cfg.Unpack(&c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *kafkaConfig) Validate() error {
	if len(c.Hosts) == 0 {
		return errors.New("no hosts configured")
	}

	if _, ok := compressionModes[strings.ToLower(c.Compression)]; !ok {
		return fmt.Errorf("compression mode '%v' unknown", c.Compression)
	}

	if err := c.Version.Validate(); err != nil {
		return err
	}

	if c.Username != "" && c.Password == "" {
		return fmt.Errorf("password must be set when username is configured")
	}

	if c.Compression == "gzip" {
		lvl := c.CompressionLevel
		if lvl != sarama.CompressionLevelDefault && !(0 <= lvl && lvl <= 9) {
			return fmt.Errorf("compression_level must be between 0 and 9")
		}
	}
	return nil
}

func newSaramaConfig(log *logp.Logger, config *kafkaConfig) (*sarama.Config, error) {
	partitioner, err := makePartitioner(log, config.Partition)
	if err != nil {
		return nil, err
	}

	k := sarama.NewConfig()

	// configure network level properties
	timeout := config.Timeout
	k.Net.DialTimeout = timeout
	k.Net.ReadTimeout = timeout
	k.Net.WriteTimeout = timeout
	k.Net.KeepAlive = config.KeepAlive
	k.Producer.Timeout = config.BrokerTimeout
	k.Producer.CompressionLevel = config.CompressionLevel

	tls, err := tlscommon.LoadTLSConfig(config.TLS)
	if err != nil {
		return nil, err
	}

	if tls != nil {
		k.Net.TLS.Enable = true
		k.Net.TLS.Config = tls.BuildModuleConfig("")
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
		}
	}

	if config.Username != "" {
		k.Net.SASL.Enable = true
		k.Net.SASL.User = config.Username
		k.Net.SASL.Password = config.Password
		err = config.Sasl.configureSarama(k)

		if err != nil {
			return nil, err
		}
	}

	// configure metadata update properties
	k.Metadata.Retry.Max = config.Metadata.Retry.Max
	k.Metadata.Retry.Backoff = config.Metadata.Retry.Backoff
	k.Metadata.RefreshFrequency = config.Metadata.RefreshFreq
	k.Metadata.Full = config.Metadata.Full

	// configure producer API properties
	if config.MaxMessageBytes != nil {
		k.Producer.MaxMessageBytes = *config.MaxMessageBytes
	}
	if config.RequiredACKs != nil {
		k.Producer.RequiredAcks = sarama.RequiredAcks(*config.RequiredACKs)
	}

	compressionMode, ok := compressionModes[strings.ToLower(config.Compression)]
	if !ok {
		return nil, fmt.Errorf("Unknown compression mode: '%v'", config.Compression)
	}
	k.Producer.Compression = compressionMode

	k.Producer.Return.Successes = true // enable return channel for signaling
	k.Producer.Return.Errors = true

	// have retries being handled by libbeat, disable retries in sarama library
	retryMax := config.MaxRetries
	if retryMax < 0 {
		retryMax = 1000
	}
	k.Producer.Retry.Max = retryMax
	k.Producer.Retry.BackoffFunc = makeBackoffFunc(config.Backoff)

	// configure per broker go channel buffering
	k.ChannelBufferSize = config.ChanBufferSize

	// configure bulk size
	k.Producer.Flush.MaxMessages = config.BulkMaxSize
	if config.BulkFlushFrequency > 0 {
		k.Producer.Flush.Frequency = config.BulkFlushFrequency
	}

	// configure client ID
	k.ClientID = config.ClientID

	version, ok := config.Version.Get()
	if !ok {
		return nil, fmt.Errorf("Unknown/unsupported kafka version: %v", config.Version)
	}
	k.Version = version

	k.Producer.Partitioner = partitioner
	k.MetricRegistry = adapter.GetGoMetrics(
		monitoring.Default,
		"libbeat.outputs.kafka",
		adapter.Rename("incoming-byte-rate", "bytes_read"),
		adapter.Rename("outgoing-byte-rate", "bytes_write"),
		adapter.GoMetricsNilify,
	)

	if err := k.Validate(); err != nil {
		log.Errorf("Invalid kafka configuration: %+v", err)
		return nil, err
	}
	return k, nil
}

// makeBackoffFunc returns a stateless implementation of exponential-backoff-with-jitter. It is conceptually
// equivalent to the stateful implementation used by other outputs, EqualJitterBackoff.
func makeBackoffFunc(cfg backoffConfig) func(retries, maxRetries int) time.Duration {
	maxBackoffRetries := int(math.Ceil(math.Log2(float64(cfg.Max) / float64(cfg.Init))))

	return func(retries, _ int) time.Duration {
		// compute 'base' duration for exponential backoff
		dur := cfg.Max
		if retries < maxBackoffRetries {
			dur = time.Duration(uint64(cfg.Init) * uint64(1<<retries))
		}

		// apply about equaly distributed jitter in second half of the interval, such that the wait
		// time falls into the interval [dur/2, dur]
		limit := int64(dur / 2)
		jitter := rand.Int63n(limit + 1)
		return time.Duration(limit + jitter)
	}
}
