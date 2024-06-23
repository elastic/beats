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

	"github.com/Shopify/sarama"
)

// Version is a kafka version
type Version string

var (
	// Sarama expects version strings to be fully expanded, e.g. "1.1.1".
	// We also allow versions to be specified as a prefix, e.g. "1",
	// understood as referencing the most recent version starting with "1".
	// truncatedKafkaVersions stores a lookup of the abbreviations we accept.
	truncatedKafkaVersions = map[string]sarama.KafkaVersion{
		"0.8.2": sarama.V0_8_2_2,
		"0.8":   sarama.V0_8_2_2,

		"0.9.0": sarama.V0_9_0_1,
		"0.9":   sarama.V0_9_0_1,

		"0.10.0": sarama.V0_10_0_1,
		"0.10.1": sarama.V0_10_1_0,
		"0.10.2": sarama.V0_10_2_1,
		"0.10":   sarama.V0_10_2_1,

		"0.11.0": sarama.V0_11_0_2,
		"0.11":   sarama.V0_11_0_2,

		"1.0": sarama.V1_0_0_0,
		"1.1": sarama.V1_1_1_0,
		"1":   sarama.V1_1_1_0,

		"2.0": sarama.V2_0_1_0,
		"2.1": sarama.V2_1_0_0,
		"2.2": sarama.V2_2_0_0,
		"2.3": sarama.V2_3_0_0,
		"2.4": sarama.V2_4_0_0,
		"2.5": sarama.V2_5_0_0,
		"2.6": sarama.V2_6_0_0,
		"2":   sarama.V2_6_0_0,
	}
)

// Validate that a kafka version is among the possible options
func (v *Version) Validate() error {
	if _, ok := v.Get(); ok {
		return nil
	}
	return fmt.Errorf("unknown/unsupported kafka version '%v'", *v)
}

// Unpack a kafka version
func (v *Version) Unpack(s string) error {
	tmp := Version(s)
	if err := tmp.Validate(); err != nil {
		return err
	}

	*v = tmp
	return nil
}

// Get a sarama kafka version
func (v Version) Get() (sarama.KafkaVersion, bool) {
	// First check if it's one of the abbreviations we accept.
	// If not, let sarama parse it.
	s := string(v)
	if version, ok := truncatedKafkaVersions[s]; ok {
		return version, true
	}
	version, err := sarama.ParseKafkaVersion(s)
	if err != nil {
		return sarama.KafkaVersion{}, false
	}
	for _, supp := range sarama.SupportedVersions {
		if version == supp {
			return version, true
		}
	}
	return sarama.KafkaVersion{}, false
}
