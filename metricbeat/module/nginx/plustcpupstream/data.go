package plustcpupstream

import (
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/module/nginx"
)

// Map body to []MapStr
func eventMapping(m *MetricSet, body io.ReadCloser, hostname string, metricset string) ([]common.MapStr, error) {
	// Nginx plus server tcpupstreams:
	b, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, err
	}

	var tcpupstreams map[string]interface{}
	if err := json.Unmarshal([]byte(b), &tcpupstreams); err != nil {
		return nil, err
	}
	tcpupstreams = nginx.Ftoi(tcpupstreams)

	events := []common.MapStr{}

	for name, tcpupstream := range tcpupstreams {
		event := common.MapStr{
			"hostname": hostname,
			"name": name,
		}

		for k, v := range tcpupstream.(map[string]interface{}) {
			event[k] = v
		}

		events = append(events, event)
	}

	return events, nil
}
