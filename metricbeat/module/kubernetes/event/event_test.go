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
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/elastic/beats/libbeat/common"
)

func TestGenerateMapStrFromEvent(t *testing.T) {
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

	expectedLabelsMapStrWithDot := common.MapStr{
		"app": common.MapStr{
			"kubernetes": common.MapStr{
				"io/version":   "5.7.21",
				"io/component": "database",
				"io/name":      "mysql",
			},
		},
	}

	expectedLabelsMapStrWithDeDot := common.MapStr{
		"app_kubernetes_io/name":      "mysql",
		"app_kubernetes_io/version":   "5.7.21",
		"app_kubernetes_io/component": "database",
	}

	expectedAnnotationsMapStrWithDot := common.MapStr{
		"prometheus": common.MapStr{
			"io/path":   "/metrics",
			"io/port":   "9102",
			"io/scheme": "http",
			"io/scrape": "false",
		},
	}

	expectedAnnotationsMapStrWithDeDot := common.MapStr{
		"prometheus_io/path":   "/metrics",
		"prometheus_io/port":   "9102",
		"prometheus_io/scheme": "http",
		"prometheus_io/scrape": "false",
	}

	testCases := map[string]struct {
		mockEvent        v1.Event
		expectedMetadata common.MapStr
		dedotConfig      dedotConfig
	}{
		"no dedots": {
			mockEvent: v1.Event{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: annotations,
				},
			},
			expectedMetadata: common.MapStr{
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
			},
			expectedMetadata: common.MapStr{
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
			},
			expectedMetadata: common.MapStr{
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
			},
			expectedMetadata: common.MapStr{
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
			mapStrOutput := generateMapStrFromEvent(&test.mockEvent, test.dedotConfig)
			assert.Equal(t, test.expectedMetadata["labels"], mapStrOutput["metadata"].(common.MapStr)["labels"])
			assert.Equal(t, test.expectedMetadata["annotations"], mapStrOutput["metadata"].(common.MapStr)["annotations"])
		})
	}
}
