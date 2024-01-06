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

package pulsar

import (
	"fmt"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"

	"github.com/elastic/beats/v7/libbeat/outputs/codec"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/go-ucfg"
)

var (
	_ ucfg.Validator = (*pulsarConfig)(nil)
)

// Config defines configuration for Pulsar exporter.
type pulsarConfig struct {
	Endpoint                   string           `config:"endpoint"`
	Timeout                    time.Duration    `config:"timeout"`
	Topic                      string           `config:"topic"`
	Producer                   *producer        `config:"producer"`
	Codec                      codec.Config     `config:"codec"`
	MaxRetries                 int              `config:"max_retries"         validate:"min=-1,nonzero"`
	TLSTrustCertsFilePath      string           `config:"tls_trust_certs_file_path"`
	TLSAllowInsecureConnection bool             `config:"tls_allow_insecure_connection"`
	Authentication             authentication   `config:"auth"`
	OperationTimeout           time.Duration    `config:"operation_timeout"`
	ConnectionTimeout          time.Duration    `config:"connection_timeout"`
	MaxConnectionsPerBroker    int              `config:"max_connections_per_broker"`
	Queue                      config.Namespace `config:"queue"`
	BulkMaxSize                int              `config:"bulk_max_size"`
}

// Validate validates the pulsar configuration.
func (c *pulsarConfig) Validate() error {
	return nil
}

// auth returns the authentication method for the pulsar client.
func (c *pulsarConfig) auth() pulsar.Authentication {
	authentication := c.Authentication
	if authentication.TLS != nil {
		return pulsar.NewAuthenticationTLS(authentication.TLS.CertFile, authentication.TLS.KeyFile)
	}
	if authentication.Token != nil {
		return pulsar.NewAuthenticationToken(authentication.Token.Token)
	}
	if authentication.OAuth2 != nil {
		return pulsar.NewAuthenticationOAuth2(map[string]string{
			"issuerUrl": authentication.OAuth2.IssuerURL,
			"clientId":  authentication.OAuth2.ClientID,
			"audience":  authentication.OAuth2.Audience,
		})
	}
	if authentication.Athenz != nil {
		return pulsar.NewAuthenticationAthenz(map[string]string{
			"providerDomain":  authentication.Athenz.ProviderDomain,
			"tenantDomain":    authentication.Athenz.TenantDomain,
			"tenantService":   authentication.Athenz.TenantService,
			"privateKey":      authentication.Athenz.PrivateKey,
			"keyId":           authentication.Athenz.KeyID,
			"principalHeader": authentication.Athenz.PrincipalHeader,
			"ztsUrl":          authentication.Athenz.ZtsURL,
		})
	}

	return nil
}

// parseConfig parses the pulsar configuration for the PulsarProducer.
func (c *pulsarConfig) parseProducerOptions() pulsar.ProducerOptions {
	if c.Producer == nil {
		return pulsar.ProducerOptions{
			Topic:       c.Topic,
			SendTimeout: c.Timeout,
		}
	}

	return pulsar.ProducerOptions{
		Topic:                           c.Topic,
		SendTimeout:                     c.Timeout,
		BatcherBuilderType:              c.Producer.parseBatchBuilderType(),
		BatchingMaxMessages:             c.Producer.BatchingMaxMessages,
		BatchingMaxPublishDelay:         c.Producer.BatchingMaxPublishDelay,
		BatchingMaxSize:                 c.Producer.BatchingMaxSize,
		CompressionLevel:                c.Producer.parseCompressionLevel(),
		CompressionType:                 c.Producer.parseCompressionType(),
		DisableBatching:                 c.Producer.DisableBatching,
		DisableBlockIfQueueFull:         c.Producer.DisableBlockIfQueueFull,
		HashingScheme:                   c.Producer.parseHashingScheme(),
		MaxPendingMessages:              c.Producer.MaxPendingMessages,
		MaxReconnectToBroker:            c.Producer.MaxReconnectToBroker,
		PartitionsAutoDiscoveryInterval: c.Producer.PartitionsAutoDiscoveryInterval,
	}
}

// parseConfig parses the pulsar configuration for the PulsarClient.
func (c *pulsarConfig) parseClientOptions() (pulsar.ClientOptions, error) {
	if len(c.Endpoint) == 0 {
		return pulsar.ClientOptions{}, fmt.Errorf("endpoint is required")

	}
	if len(c.Topic) == 0 {
		return pulsar.ClientOptions{}, fmt.Errorf("topic is required")
	}

	options := pulsar.ClientOptions{
		URL:                     c.Endpoint,
		ConnectionTimeout:       c.ConnectionTimeout,
		OperationTimeout:        c.OperationTimeout,
		MaxConnectionsPerBroker: c.MaxConnectionsPerBroker,
	}

	options.TLSAllowInsecureConnection = c.TLSAllowInsecureConnection
	if len(c.TLSTrustCertsFilePath) > 0 {
		options.TLSTrustCertsFilePath = c.TLSTrustCertsFilePath
	}

	options.Authentication = c.auth()

	return options, nil
}

// defaultConfig returns the default configuration for the Pulsar output.
func defaultConfig() *pulsarConfig {
	return &pulsarConfig{
		Endpoint: "pulsar://localhost:6650",
		// using an empty topic to track when it has not been set by user, default is based on traces or metrics.
		Topic:                      "persistent://public/default/beats",
		Authentication:             authentication{},
		Codec:                      codec.Config{},
		MaxRetries:                 3,
		BulkMaxSize:                1024,
		MaxConnectionsPerBroker:    1,
		ConnectionTimeout:          5 * time.Second,
		OperationTimeout:           30 * time.Second,
		TLSAllowInsecureConnection: false,
	}
}

// readConfig reads the configuration for the Pulsar output.
func readConfig(config *config.C) (*pulsarConfig, error) {
	cfg := defaultConfig()
	if err := config.Unpack(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// The following structs are Pulsar internal structs.
// ------------------------------------------------------------------------------------------------------------

// Authentication defines authentication configuration for Pulsar.
type authentication struct {
	TLS    *tls    `config:"tls"`
	Token  *token  `config:"token"`
	Athenz *athenz `config:"athenz"`
	OAuth2 *oauth2 `config:"oauth2"`
}

type tls struct {
	CertFile string `config:"cert_file"`
	KeyFile  string `config:"key_file"`
}

type token struct {
	Token string `config:"token"`
}

type athenz struct {
	ProviderDomain  string `config:"provider_domain"`
	TenantDomain    string `config:"tenant_domain"`
	TenantService   string `config:"tenant_service"`
	PrivateKey      string `config:"private_key"`
	KeyID           string `config:"key_id"`
	PrincipalHeader string `config:"principal_header"`
	ZtsURL          string `config:"zts_url"`
}

type oauth2 struct {
	IssuerURL string `config:"issuer_url"`
	ClientID  string `config:"client_id"`
	Audience  string `config:"audience"`
}

// Producer defines configuration for producer
type producer struct {
	MaxReconnectToBroker            *uint         `config:"max_reconnect_broker"`
	HashingScheme                   string        `config:"hashing_scheme"`
	CompressionLevel                string        `config:"compression_level"`
	CompressionType                 string        `config:"compression_type"`
	MaxPendingMessages              int           `config:"max_pending_messages"`
	BatcherBuilderType              string        `config:"batch_builder_type"`
	PartitionsAutoDiscoveryInterval time.Duration `config:"partitions_auto_discovery_interval"`
	BatchingMaxPublishDelay         time.Duration `config:"batching_max_publish_delay"`
	BatchingMaxMessages             uint          `config:"batching_max_messages"`
	BatchingMaxSize                 uint          `config:"batching_max_size"`
	DisableBlockIfQueueFull         bool          `config:"disable_block_if_queue_full"`
	DisableBatching                 bool          `config:"disable_batching"`
}

const (
	DefaultBatchBuilder  string = "default"
	KeyBasedBatchBuilder string = "key_based"
)

func (c *producer) parseBatchBuilderType() pulsar.BatcherBuilderType {
	switch c.BatcherBuilderType {
	case DefaultBatchBuilder:
		return pulsar.DefaultBatchBuilder
	case KeyBasedBatchBuilder:
		return pulsar.KeyBasedBatchBuilder
	default:
		fmt.Printf("unsupported batcher builder type: %s", c.BatcherBuilderType)
		return pulsar.DefaultBatchBuilder
	}
}

const (
	None string = "none"
	LZ4  string = "lz4"
	ZLib string = "zlib"
	ZStd string = "zstd"
)

func (c *producer) parseCompressionType() pulsar.CompressionType {
	switch c.CompressionType {
	case None:
		return pulsar.NoCompression
	case LZ4:
		return pulsar.LZ4
	case ZLib:
		return pulsar.ZLib
	case ZStd:
		return pulsar.ZSTD
	default:
		fmt.Printf("unsupported compression type: %s", c.CompressionType)
		return pulsar.NoCompression
	}
}

const (
	Default string = "default"
	Faster  string = "faster"
	Better  string = "better"
)

func (c *producer) parseCompressionLevel() pulsar.CompressionLevel {
	switch c.CompressionLevel {
	case Default:
		return pulsar.Default
	case Faster:
		return pulsar.Faster
	case Better:
		return pulsar.Better
	default:
		fmt.Printf("unsupported compression level: %s", c.CompressionLevel)
		return pulsar.Default
	}
}

const (
	JavaStringHash string = "java_string_hash"
	Murmur3_32Hash string = "murmur3_32hash"
)

func (c *producer) parseHashingScheme() pulsar.HashingScheme {
	switch c.HashingScheme {
	case JavaStringHash:
		return pulsar.JavaStringHash
	case Murmur3_32Hash:
		return pulsar.Murmur3_32Hash
	default:
		fmt.Printf("unsupported hashing scheme: %s", c.HashingScheme)
		return pulsar.JavaStringHash
	}
}
