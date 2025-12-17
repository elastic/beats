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

//go:build !integration

package server

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func GetMetricProcessor() *metricProcessor {
	templates := []TemplateConfig{
		{
			Namespace: "foo",
			Filter:    "test.localhost.*",
			Template:  ".host.shell.metric",
			Delimiter: ".",
		},
		{
			Namespace: "foo",
			Filter:    "test.xyz.*",
			Template:  ".host.metric*",
			Delimiter: "_",
			Tags: map[string]string{
				"a": "b",
			},
		},
	}

	defaultTemplate := DefaultGraphiteCollectorConfig().DefaultTemplate
	return NewMetricProcessor(templates, defaultTemplate)
}

func TestMetricProcessorAddTemplate(t *testing.T) {
	processor := GetMetricProcessor()
	temp := TemplateConfig{
		Namespace: "xyz",
		Filter:    "a.b.*",
		Template:  ".host.shell.metric",
		Delimiter: ".",
	}
	processor.AddTemplate(temp)
	out := processor.templates.Search([]string{"a", "b", "c"})
	assert.NotNil(t, out)
	assert.Equal(t, out.Namespace, temp.Namespace)
}

func TestMetricProcessorDeleteTemplate(t *testing.T) {
	processor := GetMetricProcessor()
	temp := TemplateConfig{
		Namespace: "xyz",
		Filter:    "a.b.*",
		Template:  ".host.shell.metric",
		Delimiter: ".",
	}
	processor.AddTemplate(temp)
	processor.RemoveTemplate(temp)
	out := processor.templates.Search([]string{"a", "b", "c"})
	assert.Nil(t, out)
}

func TestMetricProcessorProcess(t *testing.T) {
	processor := GetMetricProcessor()
	event, err := processor.Process("test.localhost.bash.stats 42 1500934723")
	assert.NoError(t, err)
	assert.NotNil(t, event)

	tag := event["tag"].(mapstr.M)
	assert.Equal(t, len(tag), 2)
	assert.Equal(t, tag["host"], "localhost")
	assert.Equal(t, tag["shell"], "bash")

	assert.NotNil(t, event["stats"])
	assert.Equal(t, event["stats"], float64(42))

	ts := float64(1500934723)
	timestamp := common.Time(time.Unix(int64(ts), int64((ts-math.Floor(ts))*float64(time.Second))))

	assert.Equal(t, event["@timestamp"], timestamp)

	event, err = processor.Process("test.localhost.bash.stats 42")
	assert.NoError(t, err)
	assert.NotNil(t, event)

	assert.NotNil(t, event["stats"])
	assert.Equal(t, event["stats"], float64(42))
}

func TestTemplateApply(t *testing.T) {
	tests := []struct {
		name          string
		tmpl          template
		parts         []string
		wantMetric    string
		wantTagsCount int
	}{
		{
			name: "metric shorter than template",
			tmpl: template{
				Delimiter: ".",
				Parts:     []string{"", "host", "region", "service", "metric"},
			},
			parts:         []string{"server1", "us-east"},
			wantMetric:    "",
			wantTagsCount: 1,
		},
		{
			name: "single part metric",
			tmpl: template{
				Delimiter: ".",
				Parts:     []string{"", "host", "region", "service", "metric"},
			},
			parts:         []string{"server1"},
			wantMetric:    "",
			wantTagsCount: 0,
		},
		{
			name: "empty metric parts",
			tmpl: template{
				Delimiter: ".",
				Parts:     []string{"", "host", "region", "service", "metric"},
			},
			parts:         []string{},
			wantMetric:    "",
			wantTagsCount: 0,
		},
		{
			name: "nil metric parts",
			tmpl: template{
				Delimiter: ".",
				Parts:     []string{"", "host", "region", "service", "metric"},
			},
			parts:         nil,
			wantMetric:    "",
			wantTagsCount: 0,
		},
		{
			name: "metric star captures remaining from current index",
			tmpl: template{
				Delimiter: "_",
				Parts:     []string{"", "host", "metric*"},
			},
			parts:         []string{"server1", "cpu", "idle", "percent"},
			wantMetric:    "idle_percent",
			wantTagsCount: 1,
		},
		{
			name: "empty template parts",
			tmpl: template{
				Delimiter: ".",
				Parts:     []string{},
			},
			parts:         []string{"server1", "us-east"},
			wantMetric:    "",
			wantTagsCount: 0,
		},
		{
			name: "template with predefined tags",
			tmpl: template{
				Delimiter: ".",
				Parts:     []string{"metric"},
				Tags:      map[string]string{"env": "prod", "dc": "us-east"},
			},
			parts:         []string{"cpu"},
			wantMetric:    "cpu",
			wantTagsCount: 2,
		},
		{
			name: "duplicate tag keys in template are combined",
			tmpl: template{
				Delimiter: "_",
				Parts:     []string{"host", "host", "metric"},
			},
			parts:         []string{"server1", "server2", "cpu"},
			wantMetric:    "cpu",
			wantTagsCount: 1,
		},
		{
			name: "all parts are metric",
			tmpl: template{
				Delimiter: ".",
				Parts:     []string{"metric", "metric", "metric"},
			},
			parts:         []string{"cpu", "idle", "percent"},
			wantMetric:    "cpu.idle.percent",
			wantTagsCount: 0,
		},
		{
			name: "metric star at beginning",
			tmpl: template{
				Delimiter: "_",
				Parts:     []string{"metric*"},
			},
			parts:         []string{"cpu", "idle", "percent"},
			wantMetric:    "cpu_idle_percent",
			wantTagsCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NotPanics(t, func() {
				metric, tags := tt.tmpl.Apply(tt.parts)
				assert.Equal(t, tt.wantMetric, metric)
				assert.Len(t, tags, tt.wantTagsCount)
			})
		})
	}
}
