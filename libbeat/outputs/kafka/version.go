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

import "github.com/Shopify/sarama"

// TODO: remove me.
// Compat version overwrite for missing versions in sarama
// Public API is compatible between these versions.
var (
	v0_10_2_1 = parseKafkaVersion("0.10.2.1")
	v0_11_0_1 = parseKafkaVersion("0.11.0.1")
	v0_11_0_2 = parseKafkaVersion("0.11.0.2")
	v1_0_1    = parseKafkaVersion("1.0.1")
	v1_1_0    = parseKafkaVersion("1.1.0")

	kafkaVersions = map[string]sarama.KafkaVersion{
		"": sarama.V1_0_0_0,

		"0.8.2.0": sarama.V0_8_2_0,
		"0.8.2.1": sarama.V0_8_2_1,
		"0.8.2.2": sarama.V0_8_2_2,
		"0.8.2":   sarama.V0_8_2_2,
		"0.8":     sarama.V0_8_2_2,

		"0.9.0.0": sarama.V0_9_0_0,
		"0.9.0.1": sarama.V0_9_0_1,
		"0.9.0":   sarama.V0_9_0_1,
		"0.9":     sarama.V0_9_0_1,

		"0.10.0.0": sarama.V0_10_0_0,
		"0.10.0.1": sarama.V0_10_0_1,
		"0.10.0":   sarama.V0_10_0_1,
		"0.10.1.0": sarama.V0_10_1_0,
		"0.10.1":   sarama.V0_10_1_0,
		"0.10.2.0": sarama.V0_10_2_0,
		"0.10.2.1": v0_10_2_1,
		"0.10.2":   v0_10_2_1,
		"0.10":     v0_10_2_1,

		"0.11.0.0": sarama.V0_11_0_0,
		"0.11.0.1": v0_11_0_1,
		"0.11.0.2": v0_11_0_2,
		"0.11.0":   v0_11_0_2,
		"0.11":     v0_11_0_2,

		"1.0.0": sarama.V1_0_0_0,
		"1.0.1": v1_0_1,
		"1.0":   v1_0_1,
		"1.1.0": v1_1_0,
		"1.1":   v1_1_0,
		"1":     v1_1_0,
	}
)

func parseKafkaVersion(s string) sarama.KafkaVersion {
	v, err := sarama.ParseKafkaVersion(s)
	if err != nil {
		panic(err)
	}
	return v
}
