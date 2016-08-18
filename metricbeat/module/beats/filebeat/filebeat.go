package filebeat

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
)

func init() {
	if err := mb.Registry.AddMetricSet("beats", "filebeat", New); err != nil {
		panic(err)
	}
}

type MetricSet struct {
	mb.BaseMetricSet
	client *http.Client // HTTP client that is reused across requests.
	url    string       // Httpprof endpoint URL.
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	logp.Warn("EXPERIMENTAL: The filebeat metricset is experimental")

	// Additional configuration options
	config := struct {
		VarsPath string `config:"vars_path"`
	}{
		VarsPath: "/debug/vars",
	}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	url := "http://" + base.Host() + config.VarsPath

	return &MetricSet{
		BaseMetricSet: base,
		url:           url,
		client:        &http.Client{Timeout: base.Module().Config().Timeout},
	}, nil
}

func (m *MetricSet) Fetch() (common.MapStr, error) {

	req, err := http.NewRequest("GET", m.url, nil)
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making http request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, resp.Status)
	}

	buf, _ := ioutil.ReadAll(resp.Body)
	data := map[string]interface{}{}
	json.Unmarshal(buf, &data)

	return schema.Apply(data), nil
}
