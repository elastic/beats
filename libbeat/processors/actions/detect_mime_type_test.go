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

package actions

import (
	"testing"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/elastic/beats/v7/libbeat/otel/otelmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func assertMimeTypeRunPdataEquivalent(t *testing.T, p beat.Processor, inputFields mapstr.M) {
	t.Helper()
	pp, ok := p.(interface {
		RunPdata(pcommon.Map) (bool, error)
	})
	require.True(t, ok, "processor must implement RunPdata")

	observed, err := p.Run(&beat.Event{Fields: inputFields.Clone()})
	require.NoError(t, err)

	body := pcommon.NewMap()
	require.NoError(t, otelmap.FromMapstr(body, inputFields))
	drop, err := pp.RunPdata(body)
	require.NoError(t, err)
	assert.False(t, drop)

	legacyNorm := pcommon.NewMap()
	require.NoError(t, otelmap.FromMapstr(legacyNorm, observed.Fields))
	assert.Equal(t, otelmap.ToMapstr(legacyNorm), otelmap.ToMapstr(body),
		"Run and RunPdata must produce identical output")
}

func TestMimeTypeFromTo(t *testing.T) {
	evt := beat.Event{
		Fields: mapstr.M{
			"foo.bar.baz": "hello world!",
		},
	}
	p, err := NewDetectMimeType(conf.MustNewConfigFrom(map[string]any{
		"field":  "foo.bar.baz",
		"target": "bar.baz.zoiks",
	}), logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)
	observed, err := p.Run(&evt)
	require.NoError(t, err)
	enriched, err := observed.Fields.GetValue("bar.baz.zoiks")
	require.NoError(t, err)
	require.Equal(t, "text/plain; charset=utf-8", enriched)

	// RunPdata parity.
	assertMimeTypeRunPdataEquivalent(t, p, evt.Fields)
}

func TestMimeTypeFromToMetadata(t *testing.T) {
	evt := beat.Event{
		Meta: mapstr.M{},
		Fields: mapstr.M{
			"foo.bar.baz": "hello world!",
		},
	}
	expectedMeta := mapstr.M{
		"field": "text/plain; charset=utf-8",
	}
	p, err := NewDetectMimeType(conf.MustNewConfigFrom(map[string]any{
		"field":  "foo.bar.baz",
		"target": "@metadata.field",
	}), logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	observed, err := p.Run(&evt)
	require.NoError(t, err)
	require.Equal(t, expectedMeta, observed.Meta)
	require.Equal(t, evt.Fields, observed.Fields)
}

func TestMimeTypeTestNoMatch(t *testing.T) {
	evt := beat.Event{
		Fields: mapstr.M{
			"foo.bar.baz": string([]byte{0, 0}),
		},
	}
	p, err := NewDetectMimeType(conf.MustNewConfigFrom(map[string]any{
		"field":  "foo.bar.baz",
		"target": "bar.baz.zoiks",
	}), logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)
	observed, err := p.Run(&evt)
	require.NoError(t, err)
	hasKey, _ := observed.Fields.HasKey("bar.baz.zoiks")
	require.False(t, hasKey)

	// RunPdata parity.
	assertMimeTypeRunPdataEquivalent(t, p, evt.Fields)
}
