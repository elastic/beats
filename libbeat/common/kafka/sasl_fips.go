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

//go:build requirefips

package kafka

import (
	"fmt"
	"strings"

	"github.com/elastic/sarama"
)

func (c *SaslConfig) Validate() error {
	switch strings.ToUpper(c.SaslMechanism) { // try not to force users to use all upper case
	case "", saslTypePlaintext:
	default:
		return fmt.Errorf("not valid SASL mechanism '%v', only supported with PLAIN", c.SaslMechanism)
	}
	return nil
}

func scramClient(mechanism string) func() sarama.SCRAMClient {
	// This should never happen because `SaslMechanism` is checked on `Validate()`, keeping a panic to detect it earlier if it happens.
	panic("scram sasl auth not supported in fips mode")
}
