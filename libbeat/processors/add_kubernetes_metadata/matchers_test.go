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

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/elastic/beats/v7/libbeat/otel/otelmap"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestFieldMatcher(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

	testCfg := map[string]any{
		"lookup_fields": []string{},
	}
	fieldCfg, err := config.NewConfigFrom(testCfg)

	assert.NoError(t, err)
	matcher, err := NewFieldMatcher(*fieldCfg, logger)
	assert.Error(t, err)
	assert.Nil(t, matcher)

	testCfg["lookup_fields"] = "foo"
	fieldCfg, _ = config.NewConfigFrom(testCfg)

	matcher, err = NewFieldMatcher(*fieldCfg, logger)
	assert.NotNil(t, matcher)
	assert.NoError(t, err)

	input := mapstr.M{
		"foo": "bar",
	}

	out := matcher.MetadataIndexCandidates(input)
	assert.Equal(t, []string{"bar"}, out)

	nonMatchInput := mapstr.M{
		"not": "match",
	}

	out = matcher.MetadataIndexCandidates(nonMatchInput)
	assert.Empty(t, out)

	// An empty-string field value must be skipped, not returned as a candidate.
	// Without this guard, [""] would block all subsequent matchers in the chain.
	emptyInput := mapstr.M{"foo": ""}
	out = matcher.MetadataIndexCandidates(emptyInput)
	assert.Empty(t, out, "empty-string field value must not produce a candidate")
}

func TestFieldMatcherRegex(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	testCfg := map[string]any{
		"lookup_fields": []string{"foo"},
		"regex_pattern": "(?!)",
	}
	fieldCfg, err := config.NewConfigFrom(testCfg)
	assert.NoError(t, err)
	matcher, err := NewFieldMatcher(*fieldCfg, logger)
	assert.ErrorContains(t, err, "invalid regex:")
	assert.Nil(t, matcher)

	testCfg["regex_pattern"] = "(?P<invalid>.*)"
	fieldCfg, _ = config.NewConfigFrom(testCfg)

	matcher, err = NewFieldMatcher(*fieldCfg, logger)
	assert.ErrorContains(t, err, "regex missing required capture group `key`")
	assert.Nil(t, matcher)

	testCfg["regex_pattern"] = "bar-(?P<key>[^-]+)-suffix"
	fieldCfg, _ = config.NewConfigFrom(testCfg)

	matcher, err = NewFieldMatcher(*fieldCfg, logger)
	require.NoError(t, err)
	require.NotNil(t, matcher)

	input := mapstr.M{
		"foo": "bar-keyvalue-suffix",
	}

	out := matcher.MetadataIndexCandidates(input)
	assert.Equal(t, []string{"keyvalue"}, out)

	nonMatchInput := mapstr.M{
		"not": "match",
		"foo": "nomatch",
	}

	out = matcher.MetadataIndexCandidates(nonMatchInput)
	assert.Empty(t, out)

	// MetadataIndexCandidatesPdata parity for the regex path.
	pm, ok := matcher.(pdataMatcher)
	require.True(t, ok, "FieldMatcher must implement pdataMatcher")

	matchBody := pcommon.NewMap()
	require.NoError(t, otelmap.FromMapstr(matchBody, input))
	assert.Equal(t, []string{"keyvalue"}, pm.MetadataIndexCandidatesPdata(matchBody))

	noMatchBody := pcommon.NewMap()
	require.NoError(t, otelmap.FromMapstr(noMatchBody, nonMatchInput))
	assert.Empty(t, pm.MetadataIndexCandidatesPdata(noMatchBody))
}

func TestFieldFormatMatcher(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

	testCfg := map[string]any{}
	fieldCfg, err := config.NewConfigFrom(testCfg)

	assert.NoError(t, err)
	matcher, err := NewFieldFormatMatcher(*fieldCfg, logger)
	assert.Error(t, err)
	assert.Nil(t, matcher)

	testCfg["format"] = `%{[namespace]}/%{[pod]}`
	fieldCfg, _ = config.NewConfigFrom(testCfg)

	matcher, err = NewFieldFormatMatcher(*fieldCfg, logger)
	assert.NotNil(t, matcher)
	assert.NoError(t, err)

	event := mapstr.M{
		"namespace": "foo",
		"pod":       "bar",
	}

	out := matcher.MetadataIndexCandidates(event)
	assert.Equal(t, []string{"foo/bar"}, out)

	event = mapstr.M{
		"foo": "bar",
	}
	out = matcher.MetadataIndexCandidates(event)
	assert.Empty(t, out)

	testCfg["format"] = `%{[dimensions.namespace]}/%{[dimensions.pod]}`
	fieldCfg, _ = config.NewConfigFrom(testCfg)
	matcher, err = NewFieldFormatMatcher(*fieldCfg, logger)
	assert.NotNil(t, matcher)
	assert.NoError(t, err)

	event = mapstr.M{
		"dimensions": mapstr.M{
			"pod":       "bar",
			"namespace": "foo",
		},
	}

	out = matcher.MetadataIndexCandidates(event)
	assert.Equal(t, []string{"foo/bar"}, out)
}

// TestMetadataIndexCandidatesPdataFieldMatcherParity verifies that
// MetadataIndexCandidatesPdata and MetadataIndexCandidates return the same result for
// FieldMatcher, which implements pdataMatcher.
func TestMetadataIndexCandidatesPdataFieldMatcherParity(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	cfg, err := config.NewConfigFrom(map[string]any{"lookup_fields": []string{"container.id"}})
	require.NoError(t, err)
	matcher, err := NewFieldMatcher(*cfg, logger)
	require.NoError(t, err)

	matchers := &Matchers{matchers: []Matcher{matcher}}

	t.Run("match", func(t *testing.T) {
		input := mapstr.M{"container": mapstr.M{"id": "abc123"}}
		body := pcommon.NewMap()
		require.NoError(t, otelmap.FromMapstr(body, input))
		assert.Equal(t, matcher.MetadataIndexCandidates(input), matchers.MetadataIndexCandidatesPdata(body))
	})

	t.Run("no match", func(t *testing.T) {
		input := mapstr.M{"unrelated": "field"}
		body := pcommon.NewMap()
		require.NoError(t, otelmap.FromMapstr(body, input))
		assert.Empty(t, matchers.MetadataIndexCandidatesPdata(body))
	})
}

// TestMetadataIndexCandidatesPdataFieldFormatMatcherFallback verifies that
// MetadataIndexCandidatesPdata falls back to a ToMapstr conversion for FieldFormatMatcher,
// which does not implement pdataMatcher, and still returns the correct candidates.
func TestMetadataIndexCandidatesPdataFieldFormatMatcherFallback(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	cfg, err := config.NewConfigFrom(map[string]any{"format": `%{[namespace]}/%{[pod]}`})
	require.NoError(t, err)
	matcher, err := NewFieldFormatMatcher(*cfg, logger)
	require.NoError(t, err)

	// FieldFormatMatcher must NOT implement pdataMatcher — the fallback path is what we are testing.
	_, isPdata := matcher.(pdataMatcher)
	require.False(t, isPdata, "FieldFormatMatcher must not implement pdataMatcher so the fallback is exercised")

	matchers := &Matchers{matchers: []Matcher{matcher}}
	input := mapstr.M{"namespace": "myns", "pod": "mypod"}

	body := pcommon.NewMap()
	require.NoError(t, otelmap.FromMapstr(body, input))

	assert.Equal(t, []string{"myns/mypod"}, matchers.MetadataIndexCandidatesPdata(body),
		"FieldFormatMatcher fallback must return the same candidates as MetadataIndexCandidates")
}
