package jolokia

import (
	"fmt"
	"net/http"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"io/ioutil"
	"strings"
)

var (
	debugf = logp.MakeDebug("jmx-jolokia")
)

// init registers the MetricSet with the central registry.
func init() {
	if err := mb.Registry.AddMetricSet("jmx", "jolokia", New); err != nil {
		panic(err)
	}
}

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	mb.BaseMetricSet
	client          *http.Client      // HTTP client that is reused across requests
	metricSetConfig []MetricSetConfig // array containing urls, bodies and mappings
}

// New create a new instance of the MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	// Additional configuration options
	config := struct {
		JolokiaConfigInput []MetricSetConfigInput `config:"mappings"`
	}{}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	if jolokiaConfig, parseErr := parseConfig(config.JolokiaConfigInput); parseErr != nil {
		return nil, parseErr
	} else {
		return &MetricSet{
			BaseMetricSet:   base,
			metricSetConfig: jolokiaConfig,
			client:          &http.Client{Timeout: base.Module().Config().Timeout},
		}, nil
	}

}

// Fetch methods implements the data gathering and data conversion to the right format
func (m *MetricSet) Fetch() ([]common.MapStr, error) {

	var events []common.MapStr

	for _, elem := range m.metricSetConfig {
		req, err := http.NewRequest("POST", elem.Url, strings.NewReader(elem.Body))
		resp, err := m.client.Do(req)
		if err != nil {
			fmt.Errorf("Error making http request: %#v", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			fmt.Errorf("HTTP error %d: %s", resp.StatusCode, resp.Status)
			continue
		}

		resp_body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Errorf("Error converting response body: %#v", err)
			continue
		}

		event, err := eventMapping(resp_body, elem.Mapping, elem.Application, elem.Instance)
		if err != nil {
			continue
		}

		events = append(events, event)
	}
	if events != nil {
		return events, nil
	} else {
		return nil, fmt.Errorf("No events could be fetched, please check the log for errors")
	}

}
