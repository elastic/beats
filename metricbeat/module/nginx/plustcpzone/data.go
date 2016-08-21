package plustcpzone

import (
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/module/nginx"
)

// Map body to []MapStr
func eventMapping(m *MetricSet, body io.ReadCloser, hostname string, metricset string) ([]common.MapStr, error) {
	// Nginx plus server tcpzones:
	b, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, err
	}

	var tcpzones map[string]interface{}
	if err := json.Unmarshal([]byte(b), &tcpzones); err != nil {
		return nil, err
	}
	tcpzones = nginx.Ftoi(tcpzones)

	events := []common.MapStr{}

	for name, tcpzone := range tcpzones {
		event := common.MapStr{
			"hostname": hostname,
			"name": name,
		}

		for k, v := range tcpzone.(map[string]interface{}) {
			event[k] = v
		}

		events = append(events, event)
	}

	return events, nil
}
