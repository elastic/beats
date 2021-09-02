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
)

func TestSaramaUpdate(t *testing.T) {
	// If any of these versions are considered valid by our parsing code,
	// it means someone updated sarama without updating the parsing code
	// for the new version. Gently remind them.
	flagVersions := []Version{"2.8.1", "2.9.0"}
	for _, v := range flagVersions {
		if _, ok := v.Get(); ok {
			t.Fatalf(
				"Kafka version %v is now considered valid. Did you update Sarama?\n"+
					"If so, remember to:\n"+
					"- Update truncatedKafkaVersions in libbeat/common/kafka/version.go\n"+
					"- Update the documentation to list the latest version:\n"+
					"  * libbeat/outputs/kafka/docs/kafka.asciidoc\n"+
					"  * filebeat/docs/inputs/inputs-kafka.asciidoc\n"+
					"- Update TestSaramaUpdate in libbeat/common/kafka/version_test.go\n",
				v)

		}
	}
}
