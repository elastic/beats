package plusupstream

import (
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/module/nginx"
)

// Map body to []MapStr
func eventMapping(m *MetricSet, body io.ReadCloser, hostname string, metricset string) ([]common.MapStr, error) {
	// Nginx plus server upstreams:
	b, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, err
	}

	var upstreams map[string]interface{}
	if err := json.Unmarshal([]byte(b), &upstreams); err != nil {
		return nil, err
	}
	upstreams = nginx.Ftoi(upstreams)

	events := []common.MapStr{}

	for name, upstream := range upstreams {
		event := common.MapStr{
			"hostname": hostname,
			"name": name,
		}

		for k, v := range upstream.(map[string]interface{}) {
			event[k] = v
		}

		events = append(events, event)
	}

	return events, nil
}
