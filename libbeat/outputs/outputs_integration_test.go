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

//go:build integration

package outputs_test

import (
	"fmt"
	"testing"

	"github.com/gofrs/uuid/v5"

	_ "github.com/elastic/beats/v7/libbeat/outputs/codec/json"
	_ "github.com/elastic/beats/v7/libbeat/outputs/elasticsearch"
	_ "github.com/elastic/beats/v7/libbeat/outputs/kafka"
	_ "github.com/elastic/beats/v7/libbeat/outputs/logstash"
	_ "github.com/elastic/beats/v7/libbeat/outputs/redis"
	_ "github.com/elastic/beats/v7/x-pack/libbeat/outputs/otelconsumer"
)

func TestOutputsMetricsIntegration(t *testing.T) {
	defaultEvFields := []map[string]any{
		{"msg": "message 1"},
		{"msg": "message 2"},
		{"msg": "message 3"},
		{"msg": "message 4"},
	}

	t.Run("kafka", func(t *testing.T) {
		const (
			kafkaDefaultHost = "localhost"
			kafkaDefaultPort = "9094"
		)

		kafkaHost := fmt.Sprintf("%v:%v",
			getenv("KAFKA_HOST", kafkaDefaultHost),
			getenv("KAFKA_PORT", kafkaDefaultPort),
		)
		testTopic := fmt.Sprintf("test-libbeat-%s",
			uuid.Must(uuid.NewV4()).String())

		rawCfg := map[string]interface{}{
			"hosts":   []string{kafkaHost},
			"topic":   testTopic,
			"timeout": "1s",
		}

		testOutputMetrics(t, "kafka", rawCfg, defaultEvFields)
	})

	t.Run("logstash", func(t *testing.T) {
		const (
			logstashDefaultHost     = "localhost"
			logstashTestDefaultPort = "5044"
		)

		rawCfg := map[string]interface{}{
			"hosts": []string{fmt.Sprintf("%v:%v",
				getenv("LS_HOST", logstashDefaultHost),
				getenv("LS_TCP_PORT", logstashTestDefaultPort),
			)},
			"index":      "logstash-test",
			"pipelining": 0,
		}

		testOutputMetrics(t, "logstash", rawCfg, defaultEvFields)
	})

	t.Run("redis", func(t *testing.T) {
		const (
			RedisDefaultHost = "localhost"
			RedisDefaultPort = "6379"
		)

		rawCfg := map[string]interface{}{
			"hosts": []string{fmt.Sprintf("%v:%v",
				getenv("REDIS_HOST", RedisDefaultHost),
				getenv("REDIS_PORT", RedisDefaultPort))},
			"key":      "test_publish_tcp",
			"db":       0,
			"datatype": "list",
			"timeout":  "5s",
		}

		testOutputMetrics(t, "redis", rawCfg, defaultEvFields)
	})
}
