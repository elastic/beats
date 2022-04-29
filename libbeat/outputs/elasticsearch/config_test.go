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

package elasticsearch

import (
	"testing"

	"github.com/stretchr/testify/assert"

	conf "github.com/elastic/elastic-agent-libs/config"
)

func TestValidDropPolicyConfig(t *testing.T) {
	config := `
non_indexable_policy.drop: ~
`
	c := conf.MustNewConfigFrom(config)
	elasticsearchOutputConfig, err := readConfig(c)
	if err != nil {
		t.Fatalf("Can't create test configuration from valid input")
	}
	policy, err := newNonIndexablePolicy(elasticsearchOutputConfig.NonIndexablePolicy)
	if err != nil {
		t.Fatalf("Can't create test configuration from valid input")
	}
	assert.Equal(t, drop, policy.action(), "action should be drop")
}

func TestDeadLetterIndexPolicyConfig(t *testing.T) {
	config := `
non_indexable_policy.dead_letter_index:
    index: "my-dead-letter-index"
`
	c := conf.MustNewConfigFrom(config)
	elasticsearchOutputConfig, err := readConfig(c)
	if err != nil {
		t.Fatalf("Can't create test configuration from valid input")
	}
	policy, err := newNonIndexablePolicy(elasticsearchOutputConfig.NonIndexablePolicy)
	if err != nil {
		t.Fatalf("Can't create test configuration from valid input")
	}
	assert.Equal(t, "my-dead-letter-index", policy.index(), "index should match config")
}

func TestInvalidNonIndexablePolicyConfig(t *testing.T) {
	tests := map[string]string{
		"non_indexable_policy with invalid policy": `
non_indexable_policy.juggle: ~
`,
		"dead_Letter_index policy without properties": `
non_indexable_policy.dead_letter_index: ~
`,
		"dead_Letter_index policy without index": `
non_indexable_policy.dead_letter_index:
    foo: "bar"
`,
		"dead_Letter_index policy nil index": `
non_indexable_policy.dead_letter_index:
    index: ~
`,
		"dead_Letter_index policy empty index": `
non_indexable_policy.dead_letter_index:
    index: ""
`,
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			c := conf.MustNewConfigFrom(test)
			elasticsearchOutputConfig, err := readConfig(c)
			if err != nil {
				t.Fatalf("Can't create test configuration from valid input")
			}
			_, err = newNonIndexablePolicy(elasticsearchOutputConfig.NonIndexablePolicy)
			if err == nil {
				t.Fatalf("Can create test configuration from invalid input")
			}
			t.Logf("error %s", err.Error())
		})
	}
}

func readConfig(cfg *conf.C) (*elasticsearchConfig, error) {
	c := defaultConfig
	if err := cfg.Unpack(&c); err != nil {
		return nil, err
	}
	return &c, nil
}
