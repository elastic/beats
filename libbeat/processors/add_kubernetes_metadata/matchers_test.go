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

package add_kubernetes_metadata

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestFieldMatcher(t *testing.T) {
	testCfg := map[string]interface{}{
		"lookup_fields": []string{},
	}
	fieldCfg, err := config.NewConfigFrom(testCfg)

	assert.NoError(t, err)
	matcher, err := NewFieldMatcher(*fieldCfg)
	assert.Error(t, err)

	testCfg["lookup_fields"] = "foo"
	fieldCfg, _ = config.NewConfigFrom(testCfg)

	matcher, err = NewFieldMatcher(*fieldCfg)
	assert.NotNil(t, matcher)
	assert.NoError(t, err)

	input := mapstr.M{
		"foo": "bar",
	}

	out := matcher.MetadataIndex(input)
	assert.Equal(t, out, "bar")

	nonMatchInput := mapstr.M{
		"not": "match",
	}

	out = matcher.MetadataIndex(nonMatchInput)
	assert.Equal(t, out, "")
}

func TestFieldFormatMatcher(t *testing.T) {
	testCfg := map[string]interface{}{}
	fieldCfg, err := config.NewConfigFrom(testCfg)

	assert.NoError(t, err)
	matcher, err := NewFieldFormatMatcher(*fieldCfg)
	assert.Error(t, err)

	testCfg["format"] = `%{[namespace]}/%{[pod]}`
	fieldCfg, _ = config.NewConfigFrom(testCfg)

	matcher, err = NewFieldFormatMatcher(*fieldCfg)
	assert.NotNil(t, matcher)
	assert.NoError(t, err)

	event := mapstr.M{
		"namespace": "foo",
		"pod":       "bar",
	}

	out := matcher.MetadataIndex(event)
	assert.Equal(t, "foo/bar", out)

	event = mapstr.M{
		"foo": "bar",
	}
	out = matcher.MetadataIndex(event)
	assert.Empty(t, out)

	testCfg["format"] = `%{[dimensions.namespace]}/%{[dimensions.pod]}`
	fieldCfg, _ = config.NewConfigFrom(testCfg)
	matcher, err = NewFieldFormatMatcher(*fieldCfg)
	assert.NotNil(t, matcher)
	assert.NoError(t, err)

	event = mapstr.M{
		"dimensions": mapstr.M{
			"pod":       "bar",
			"namespace": "foo",
		},
	}

	out = matcher.MetadataIndex(event)
	assert.Equal(t, "foo/bar", out)
}
