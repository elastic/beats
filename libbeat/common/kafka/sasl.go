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
	"strings"

	"github.com/elastic/sarama"
)

type SaslConfig struct {
	SaslMechanism string `config:"mechanism"`
}

const (
	saslTypePlaintext   = sarama.SASLTypePlaintext
	saslTypeSCRAMSHA256 = sarama.SASLTypeSCRAMSHA256
	saslTypeSCRAMSHA512 = sarama.SASLTypeSCRAMSHA512
	saslTypeOauthBearer = sarama.SASLTypeOAuth
)

func (c *SaslConfig) Validate() error {
	switch strings.ToUpper(c.SaslMechanism) { // try not to force users to use all upper case
	case "", saslTypePlaintext, saslTypeSCRAMSHA256, saslTypeSCRAMSHA512, saslTypeOauthBearer:
	default:
		return fmt.Errorf("not valid SASL mechanism '%v', only supported with PLAIN|SCRAM-SHA-512|SCRAM-SHA-256|OAUTHBEARER", c.SaslMechanism)
	}
	return nil
}

// ValidateWithUsername checks cross-field invariants that require knowledge of the
// configured credentials. Call this from any outer config Validate() that owns
// both the SASL mechanism and the username field.
func (c *SaslConfig) ValidateWithUsername(username string) error {
	if strings.ToUpper(c.SaslMechanism) == saslTypeOauthBearer && username != "" {
		return fmt.Errorf("sasl.mechanism OAUTHBEARER does not use username/password; remove the username/password fields or switch to PLAIN, SCRAM-SHA-256, or SCRAM-SHA-512")
	}
	return nil
}

func (c *SaslConfig) ConfigureSarama(config *sarama.Config) {
	switch strings.ToUpper(c.SaslMechanism) { // try not to force users to use all upper case
	case "":
		// SASL is not enabled
		return
	case saslTypePlaintext:
		config.Net.SASL.Mechanism = sarama.SASLMechanism(sarama.SASLTypePlaintext)
	case saslTypeSCRAMSHA256:
		config.Net.SASL.Handshake = true
		config.Net.SASL.Mechanism = sarama.SASLMechanism(sarama.SASLTypeSCRAMSHA256)
		config.Net.SASL.SCRAMClientGeneratorFunc = scramClient(saslTypeSCRAMSHA256)
	case saslTypeSCRAMSHA512:
		config.Net.SASL.Handshake = true
		config.Net.SASL.Mechanism = sarama.SASLMechanism(sarama.SASLTypeSCRAMSHA512)
		config.Net.SASL.SCRAMClientGeneratorFunc = scramClient(saslTypeSCRAMSHA512)
	default:
	}
}
