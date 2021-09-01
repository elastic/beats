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
	"github.com/elastic/beats/v7/libbeat/logp"
	"strings"

	"github.com/Shopify/sarama"

)
var debugf = logp.MakeDebug("kafka")
type SaslConfig struct {
    UserName           string `config:"username"`
	PassWord           string `config:"password"`
	SaslMechanism      string `config:"mechanism"`
	ServiceName        string `config:"serviceName"`
	Realm              string `config:"realm"`
	KerberosConfigPath string `config:"kerberosConfigPath"`
	KerberosAuthType   string `config:"kerberosAuthType"`
	KeyTabPath         string `config:"keyTabPath"`
}

const (
	saslTypePlaintext   = sarama.SASLTypePlaintext
	saslTypeSCRAMSHA256 = sarama.SASLTypeSCRAMSHA256
	saslTypeSCRAMSHA512 = sarama.SASLTypeSCRAMSHA512
	saslTypeGSSAPI      = sarama.SASLTypeGSSAPI
)

func (c *SaslConfig) ConfigureSarama(config *sarama.Config) {
	switch strings.ToUpper(c.SaslMechanism) { // try not to force users to use all upper case
	case "":
		// SASL is not enabled
		return
	case saslTypePlaintext:
		config.Net.SASL.User = c.UserName
		config.Net.SASL.Password = c.PassWord
		config.Net.SASL.Mechanism = sarama.SASLMechanism(sarama.SASLTypePlaintext)
	case saslTypeSCRAMSHA256:
		config.Net.SASL.Handshake = true
		config.Net.SASL.User = c.UserName
		config.Net.SASL.Password = c.PassWord
		config.Net.SASL.Mechanism = sarama.SASLMechanism(sarama.SASLTypeSCRAMSHA256)
		config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
			return &XDGSCRAMClient{HashGeneratorFcn: SHA256}
		}
	case saslTypeSCRAMSHA512:
		config.Net.SASL.Handshake = true
		config.Net.SASL.User = c.UserName
		config.Net.SASL.Password = c.PassWord
		config.Net.SASL.Mechanism = sarama.SASLMechanism(sarama.SASLTypeSCRAMSHA512)
		config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
			return &XDGSCRAMClient{HashGeneratorFcn: SHA512}
		}
	case saslTypeGSSAPI:
		config.Net.SASL.Mechanism = sarama.SASLMechanism(sarama.SASLTypeGSSAPI)
		config.Net.SASL.GSSAPI.ServiceName = c.ServiceName
		config.Net.SASL.GSSAPI.KerberosConfigPath = c.KerberosConfigPath
		config.Net.SASL.GSSAPI.Realm = c.Realm
		config.Net.SASL.GSSAPI.Username = c.UserName
		if c.KerberosAuthType == "keytabAuth" {
			config.Net.SASL.GSSAPI.AuthType = sarama.KRB5_KEYTAB_AUTH
			config.Net.SASL.GSSAPI.KeyTabPath = c.KeyTabPath
		} else {
			config.Net.SASL.GSSAPI.AuthType = sarama.KRB5_USER_AUTH
			config.Net.SASL.GSSAPI.Password = c.PassWord
		}
	default:
		// This should never happen because `SaslMechanism` is checked on `Validate()`, keeping a panic to detect it earlier if it happens.
		panic(fmt.Sprintf("not valid SASL mechanism '%v', only supported with PLAIN|SCRAM-SHA-512|SCRAM-SHA-256|GSSAPI", c.SaslMechanism))
	}
}

func (c *SaslConfig) Validate() error {
	switch strings.ToUpper(c.SaslMechanism) { // try not to force users to use all upper case
	case "", saslTypePlaintext, saslTypeSCRAMSHA256, saslTypeSCRAMSHA512, saslTypeGSSAPI:
	default:
		return fmt.Errorf("not valid SASL mechanism '%v', only supported with PLAIN|SCRAM-SHA-512|SCRAM-SHA-256|GSSAPI", c.SaslMechanism)
	}
	return nil
}
