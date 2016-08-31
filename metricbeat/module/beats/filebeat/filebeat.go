package filebeat

import (
	"net/http"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/beats"
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

	data, err := beats.Request(m.url, m.client)
	if err != nil {
		return nil, err
	}

	return schema.Apply(data), nil
}
