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

package event

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestGenerateMapStrFromEvent(t *testing.T) {
	logger := logp.NewLogger("kubernetes.event")

	labels := map[string]string{
		"app.kubernetes.io/name":      "mysql",
		"app.kubernetes.io/version":   "5.7.21",
		"app.kubernetes.io/component": "database",
	}

	annotations := map[string]string{
		"prometheus.io/path":   "/metrics",
		"prometheus.io/port":   "9102",
		"prometheus.io/scheme": "http",
		"prometheus.io/scrape": "false",
	}

	expectedLabelsMapStrWithDot := mapstr.M{
		"app": mapstr.M{
			"kubernetes": mapstr.M{
				"io/version":   "5.7.21",
				"io/component": "database",
				"io/name":      "mysql",
			},
		},
	}

	expectedLabelsMapStrWithDeDot := mapstr.M{
		"app_kubernetes_io/name":      "mysql",
		"app_kubernetes_io/version":   "5.7.21",
		"app_kubernetes_io/component": "database",
	}

	expectedAnnotationsMapStrWithDot := mapstr.M{
		"prometheus": mapstr.M{
			"io/path":   "/metrics",
			"io/port":   "9102",
			"io/scheme": "http",
			"io/scrape": "false",
		},
	}

	expectedAnnotationsMapStrWithDeDot := mapstr.M{
		"prometheus_io/path":   "/metrics",
		"prometheus_io/port":   "9102",
		"prometheus_io/scheme": "http",
		"prometheus_io/scrape": "false",
	}

	source := v1.EventSource{
		Component: "kubelet",
		Host:      "prod_1",
	}

	testCases := map[string]struct {
		mockEvent        v1.Event
		expectedMetadata mapstr.M
		dedotConfig      dedotConfig
	}{
		"no dedots": {
			mockEvent: v1.Event{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: annotations,
				},
				Source: source,
			},
			expectedMetadata: mapstr.M{
				"labels":      expectedLabelsMapStrWithDot,
				"annotations": expectedAnnotationsMapStrWithDot,
			},
			dedotConfig: dedotConfig{
				LabelsDedot:      false,
				AnnotationsDedot: false,
			},
		},
		"dedot labels": {
			mockEvent: v1.Event{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: annotations,
				},
				Source: source,
			},
			expectedMetadata: mapstr.M{
				"labels":      expectedLabelsMapStrWithDeDot,
				"annotations": expectedAnnotationsMapStrWithDot,
			},
			dedotConfig: dedotConfig{
				LabelsDedot:      true,
				AnnotationsDedot: false,
			},
		},
		"dedot annotatoins": {
			mockEvent: v1.Event{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: annotations,
				},
				Source: source,
			},
			expectedMetadata: mapstr.M{
				"labels":      expectedLabelsMapStrWithDot,
				"annotations": expectedAnnotationsMapStrWithDeDot,
			},
			dedotConfig: dedotConfig{
				LabelsDedot:      false,
				AnnotationsDedot: true,
			},
		},
		"dedot both labels and annotations": {
			mockEvent: v1.Event{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: annotations,
				},
				Source: source,
			},
			expectedMetadata: mapstr.M{
				"labels":      expectedLabelsMapStrWithDeDot,
				"annotations": expectedAnnotationsMapStrWithDeDot,
			},
			dedotConfig: dedotConfig{
				LabelsDedot:      true,
				AnnotationsDedot: true,
			},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			mapStrOutput := generateMapStrFromEvent(&test.mockEvent, test.dedotConfig, logger)
			assert.Equal(t, test.expectedMetadata["labels"], mapStrOutput["metadata"].(mapstr.M)["labels"])
			assert.Equal(t, test.expectedMetadata["annotations"], mapStrOutput["metadata"].(mapstr.M)["annotations"])
			assert.Equal(t, source.Host, mapStrOutput["source"].(mapstr.M)["host"])
			assert.Equal(t, source.Component, mapStrOutput["source"].(mapstr.M)["component"])
		})
	}
}
