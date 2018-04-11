package pool

import (
	"encoding/json"
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

// init registers the MetricSet with the central registry.
func init() {
	mb.Registry.MustAddMetricSet("php_fpm", "pool", New,
		mb.WithHostParser(hostParser),
		mb.DefaultMetricSet(),
	)
}

const (
	defaultScheme = "http"
	defaultPath   = "/status"
)

// hostParser is used for parsing the configured php-fpm hosts.
var hostParser = parse.URLHostParserBuilder{
	DefaultScheme: defaultScheme,
	DefaultPath:   defaultPath,
	QueryParams:   "json",
	PathConfigKey: "status_path",
}.Build()

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	mb.BaseMetricSet
	*helper.HTTP
}

// New create a new instance of the MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The php_fpm pool metricset is beta")

	http, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}
	return &MetricSet{
		base,
		http,
	}, nil
}

// Fetch gathers data for the pool metricset
func (m *MetricSet) Fetch() (common.MapStr, error) {
	content, err := m.HTTP.FetchContent()
	if err != nil {
		return nil, err
	}

	var stats map[string]interface{}
	err = json.Unmarshal(content, &stats)
	if err != nil {
		return nil, fmt.Errorf("error parsing json: %v", err)
	}

	data, _ := schema.Apply(stats)
	return data, nil
}
