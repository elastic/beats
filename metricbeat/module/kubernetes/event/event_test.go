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

	"github.com/ericchiang/k8s/apis/core/v1"
	k8s_io_apimachinery_pkg_apis_meta_v1 "github.com/ericchiang/k8s/apis/meta/v1"
	"github.com/stretchr/testify/assert"

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

	mockEvent := v1.Event{
		Metadata: &k8s_io_apimachinery_pkg_apis_meta_v1.ObjectMeta{
			Labels:      labels,
			Annotations: annotations,
		},
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

	dedotConfig1 := dedotConfig{
		LabelsDedot:      false,
		AnnotationsDedot: false,
	}
	mapStrOutput1 := generateMapStrFromEvent(&mockEvent, dedotConfig1)
	metadata1 := mapStrOutput1["metadata"].(common.MapStr)
	assert.Equal(t, expectedLabelsMapStrWithDot, metadata1["labels"])
	assert.Equal(t, expectedAnnotationsMapStrWithDot, metadata1["annotations"])

	dedotConfig2 := dedotConfig{
		LabelsDedot:      true,
		AnnotationsDedot: false,
	}
	mapStrOutput2 := generateMapStrFromEvent(&mockEvent, dedotConfig2)
	metadata2 := mapStrOutput2["metadata"].(common.MapStr)
	assert.Equal(t, expectedLabelsMapStrWithDeDot, metadata2["labels"])
	assert.Equal(t, expectedAnnotationsMapStrWithDot, metadata2["annotations"])

	dedotConfig3 := dedotConfig{
		LabelsDedot:      false,
		AnnotationsDedot: true,
	}
	mapStrOutput3 := generateMapStrFromEvent(&mockEvent, dedotConfig3)
	metadata3 := mapStrOutput3["metadata"].(common.MapStr)
	assert.Equal(t, expectedLabelsMapStrWithDot, metadata3["labels"])
	assert.Equal(t, expectedAnnotationsMapStrWithDeDot, metadata3["annotations"])

	dedotConfig4 := dedotConfig{
		LabelsDedot:      true,
		AnnotationsDedot: true,
	}
	mapStrOutput4 := generateMapStrFromEvent(&mockEvent, dedotConfig4)
	metadata4 := mapStrOutput4["metadata"].(common.MapStr)
	assert.Equal(t, expectedLabelsMapStrWithDeDot, metadata4["labels"])
	assert.Equal(t, expectedAnnotationsMapStrWithDeDot, metadata4["annotations"])
}
