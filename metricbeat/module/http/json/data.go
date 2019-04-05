package json

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"net/http"
	"strconv"
	"strings"
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
