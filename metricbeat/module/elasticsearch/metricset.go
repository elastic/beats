package elasticsearch

import (
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

const (
	defaultScheme = "http"
	pathConfigKey = "path"
)

var (
	// HostParser parses host urls for RabbitMQ management plugin
	HostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		PathConfigKey: pathConfigKey,
	}.Build()
)

// MetricSet can be used to build other metric sets that query RabbitMQ
// management plugin
type MetricSet struct {
	mb.BaseMetricSet
	*helper.HTTP
	XPack bool
}

// NewMetricSet creates an metric set that can be used to build other metric
// sets that query RabbitMQ management plugin
func NewMetricSet(base mb.BaseMetricSet, subPath string) (*MetricSet, error) {
	http, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}
	http.SetURI(http.GetURI() + subPath)

	config := struct {
		XPack bool `config:"xpack.enabled"`
	}{
		XPack: false,
	}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	if config.XPack {
		cfgwarn.Experimental("The experimental xpack.enabled flag in elasticsearch/node_stats metricset is enabled.")
	}

	return &MetricSet{
		base,
		http,
		config.XPack,
	}, nil
}
