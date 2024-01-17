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
	"testing"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"gotest.tools/assert"

	"github.com/elastic/beats/v7/libbeat/outputs/codec"
	"github.com/elastic/elastic-agent-libs/config"
)

func Test_DefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	assert.Assert(t, cfg != nil)
	assert.Equal(t, cfg.Endpoint, "pulsar://localhost:6650")
	assert.Equal(t, cfg.Topic, "persistent://public/default/beats")
	assert.Equal(t, cfg.MaxRetries, 3)
	assert.Equal(t, cfg.BulkMaxSize, 1024)
	assert.Equal(t, cfg.MaxConnectionsPerBroker, 1)
	assert.Equal(t, cfg.ConnectionTimeout, 5*time.Second)
	assert.Equal(t, cfg.OperationTimeout, 30*time.Second)
	assert.Equal(t, cfg.Codec, codec.Config{})
	assert.Equal(t, cfg.Authentication, authentication{})
}

func Test_DefaultConfigWithAuth(t *testing.T) {

	cfgMap := map[string]interface{}{
		"endpoint":                   "pulsar://localhost:6650",
		"topic":                      "persistent://public/default/beats",
		"max_retries":                3,
		"bulk_max_size":              1024,
		"timeout":                    30 * time.Second,
		"max_connections_per_broker": 1,
		"connection_timeout":         5 * time.Second,
		"operation_timeout":          30 * time.Second,
		"auth": map[string]interface{}{
			"tls": map[string]interface{}{
				"cert_file": "cert_file",
				"key_file":  "key_file",
			},
		},
	}
	cfg0 := config.MustNewConfigFrom(cfgMap)
	cfg, err := readConfig(cfg0)
	assert.NilError(t, err)

	assert.Assert(t, cfg != nil)
	assert.Equal(t, cfg.Endpoint, "pulsar://localhost:6650")
	assert.Equal(t, cfg.Topic, "persistent://public/default/beats")
	assert.Equal(t, cfg.MaxRetries, 3)
	assert.Equal(t, cfg.Timeout, 30*time.Second)
	assert.Equal(t, cfg.BulkMaxSize, 1024)
	assert.Equal(t, cfg.MaxConnectionsPerBroker, 1)
	assert.Equal(t, cfg.ConnectionTimeout, 5*time.Second)
	assert.Equal(t, cfg.OperationTimeout, 30*time.Second)
	assert.Equal(t, cfg.Codec, codec.Config{})
	assert.Equal(t, cfg.Authentication.TLS.CertFile, "cert_file")
	assert.Equal(t, cfg.Authentication.TLS.KeyFile, "key_file")
}

func Test_ProducerConfig(t *testing.T) {
	cfgMap := map[string]interface{}{
		"producer": map[string]interface{}{
			"max_reconnect_broker":               3,
			"hashing_scheme":                     "java_string_hash",
			"compression_level":                  "default",
			"compression_type":                   "lz4",
			"max_pending_messages":               1000,
			"batch_builder_type":                 "default",
			"partitions_auto_discovery_interval": 10 * time.Second,
			"batching_max_publish_delay":         10 * time.Second,
			"batching_max_messages":              100,
			"batching_max_size":                  1024,
			"disable_block_if_queue_full":        false,
			"disable_batching":                   false,
		},
	}
	cfg0 := config.MustNewConfigFrom(cfgMap)
	cfg, err := readConfig(cfg0)
	assert.NilError(t, err)

	assert.Equal(t, *cfg.Producer.MaxReconnectToBroker, uint(3))
	assert.Equal(t, cfg.Producer.HashingScheme, JavaStringHash)
	assert.Equal(t, cfg.Producer.CompressionLevel, Default)
	assert.Equal(t, cfg.Producer.CompressionType, LZ4)
	assert.Equal(t, cfg.Producer.MaxPendingMessages, 1000)
	assert.Equal(t, cfg.Producer.BatcherBuilderType, DefaultBatchBuilder)
	assert.Equal(t, cfg.Producer.PartitionsAutoDiscoveryInterval, 10*time.Second)
	assert.Equal(t, cfg.Producer.BatchingMaxPublishDelay, 10*time.Second)
	assert.Equal(t, cfg.Producer.BatchingMaxMessages, uint(100))
	assert.Equal(t, cfg.Producer.BatchingMaxSize, uint(1024))
	assert.Equal(t, cfg.Producer.DisableBlockIfQueueFull, false)
	assert.Equal(t, cfg.Producer.DisableBatching, false)
}

func Test_ProducerOptions(t *testing.T) {
	cfgMap := map[string]interface{}{
		"producer": map[string]interface{}{
			"max_reconnect_broker":               3,
			"hashing_scheme":                     "java_string_hash",
			"compression_level":                  "default",
			"compression_type":                   "lz4",
			"max_pending_messages":               1000,
			"batch_builder_type":                 "default",
			"partitions_auto_discovery_interval": 10 * time.Second,
			"batching_max_publish_delay":         10 * time.Second,
			"batching_max_messages":              100,
			"batching_max_size":                  1024,
			"disable_block_if_queue_full":        false,
			"disable_batching":                   false,
		},
	}
	cfg0 := config.MustNewConfigFrom(cfgMap)
	cfg, err := readConfig(cfg0)
	assert.NilError(t, err)

	po := cfg.parseProducerOptions()
	assert.Equal(t, *po.MaxReconnectToBroker, uint(3))
	assert.Equal(t, po.HashingScheme, pulsar.JavaStringHash)
	assert.Equal(t, po.CompressionLevel, pulsar.Default)
	assert.Equal(t, po.CompressionType, pulsar.LZ4)
	assert.Equal(t, po.MaxPendingMessages, 1000)
	assert.Equal(t, po.BatcherBuilderType, pulsar.DefaultBatchBuilder)
	assert.Equal(t, po.PartitionsAutoDiscoveryInterval, 10*time.Second)
	assert.Equal(t, po.BatchingMaxPublishDelay, 10*time.Second)
	assert.Equal(t, po.BatchingMaxMessages, uint(100))
	assert.Equal(t, po.BatchingMaxSize, uint(1024))
	assert.Equal(t, po.DisableBlockIfQueueFull, false)
	assert.Equal(t, po.DisableBatching, false)
}

func Test_ClientOptions(t *testing.T) {
	cfgMap := map[string]interface{}{
		"endpoint":                   "pulsar://localhost:6650",
		"topic":                      "persistent://public/default/beats",
		"max_connections_per_broker": 1,
		"connection_timeout":         5 * time.Second,
		"operation_timeout":          30 * time.Second,
		"auth": map[string]interface{}{
			"tls": map[string]interface{}{
				"cert_file": "cert_file",
				"key_file":  "key_file",
			},
		},
	}
	cfg0 := config.MustNewConfigFrom(cfgMap)
	cfg, err := readConfig(cfg0)
	assert.NilError(t, err)

	co, err := cfg.parseClientOptions()
	assert.NilError(t, err)
	assert.Equal(t, co.URL, "pulsar://localhost:6650")
	assert.Equal(t, co.MaxConnectionsPerBroker, 1)
	assert.Equal(t, co.ConnectionTimeout, 5*time.Second)
	assert.Equal(t, co.OperationTimeout, 30*time.Second)
	//assert.Equal(t, co.Authentication, pulsar.NewAuthenticationTLS("cert_file", "key_file"))
}
