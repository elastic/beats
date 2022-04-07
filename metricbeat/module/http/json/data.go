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

package json

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/metricbeat/mb"
)

func (m *MetricSet) processBody(response *http.Response, jsonBody interface{}) mb.Event {
	var event common.MapStr

	if m.deDotEnabled {
		event = common.DeDotJSON(jsonBody).(common.MapStr)
	} else {
		event = jsonBody.(common.MapStr)
	}

	if m.requestEnabled {
		event[mb.ModuleDataKey] = common.MapStr{
			"request": common.MapStr{
				"headers": m.getHeaders(response.Request.Header),
				"method":  response.Request.Method,
				"body": common.MapStr{
					"content": m.body,
				},
			},
		}
	}

	if m.responseEnabled {
		phrase := strings.TrimPrefix(response.Status, strconv.Itoa(response.StatusCode)+" ")
		event[mb.ModuleDataKey] = common.MapStr{
			"response": common.MapStr{
				"code":    response.StatusCode,
				"phrase":  phrase,
				"headers": m.getHeaders(response.Header),
			},
		}
	}

	return mb.Event{
		MetricSetFields: event,
		Namespace:       "http." + m.namespace,
	}
}

func (m *MetricSet) getHeaders(header http.Header) map[string]string {
	headers := make(map[string]string)
	for k, v := range header {
		value := ""
		for _, h := range v {
			value += h + " ,"
		}
		value = strings.TrimRight(value, " ,")
		headers[k] = value
	}
	return headers
}
