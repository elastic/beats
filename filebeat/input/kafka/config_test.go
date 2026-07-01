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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common/kafka"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/sarama"
)

func TestNewSaramaConfigOAUTHBEARER(t *testing.T) {
	logger := logp.NewNopLogger()

	baseConfig := func() kafkaInputConfig {
		cfg := defaultConfig()
		cfg.Hosts = []string{"localhost:9092"}
		cfg.Topics = []string{"foo"}
		cfg.GroupID = "filebeat"
		return cfg
	}

	t.Run("valid config enables SASL and sets a token provider", func(t *testing.T) {
		cfg := baseConfig()
		cfg.Sasl = kafka.SaslConfig{
			SaslMechanism:   "OAUTHBEARER",
			CredentialsPath: "/var/run/secrets/tokens/kafka.jwt",
			Extensions: map[string]string{
				"logicalCluster": "lkc-abc123",
				"identityPoolId": "pool-xyz789",
			},
		}

		sc, err := newSaramaConfig(cfg, logger)
		require.NoError(t, err)
		assert.True(t, sc.Net.SASL.Enable, "SASL should be enabled")
		assert.Equal(t, sarama.SASLMechanism(sarama.SASLTypeOAuth), sc.Net.SASL.Mechanism)
		assert.NotNil(t, sc.Net.SASL.TokenProvider, "token provider should be set for OAUTHBEARER")
	})

	t.Run("missing credentials_path is an error", func(t *testing.T) {
		cfg := baseConfig()
		cfg.Sasl = kafka.SaslConfig{SaslMechanism: "OAUTHBEARER"}

		_, err := newSaramaConfig(cfg, logger)
		require.Error(t, err, "expected an error when sasl.credentials_path is not set for OAUTHBEARER")
	})
}
