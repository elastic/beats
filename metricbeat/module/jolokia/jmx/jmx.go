package jmx

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

var (
	metricsetName = "jolokia.jmx"
	logPrefix     = "[" + metricsetName + "]"
	debugf        = logp.MakeDebug(metricsetName)
)

// init registers the MetricSet with the central registry.
func init() {
	if err := mb.Registry.AddMetricSet("jolokia", "jmx", New, hostParser); err != nil {
		panic(err)
	}
}

const (
	// defaultScheme is the default scheme to use when it is not specified in
	// the host config.
	defaultScheme = "http"

	// defaultPath is the default path to the ngx_http_stub_status_module endpoint on Nginx.
	defaultPath = "/jolokia/?ignoreErrors=true&canonicalNaming=false"
)

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		PathConfigKey: "path",
		DefaultPath:   defaultPath,
	}.Build()
)

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	mb.BaseMetricSet
	mapping   map[string]string
	namespace string
	http      *helper.HTTP
}

// New create a new instance of the MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	logp.Beta("The jolokia jmx metricset is beta")

	// Additional configuration options
	config := struct {
		Namespace string       `config:"namespace" validate:"required"`
		Mappings  []JMXMapping `config:"jmx.mappings" validate:"required"`
	}{}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	body, mapping, err := buildRequestBodyAndMapping(config.Mappings)
	if err != nil {
		return nil, err
	}

	http := helper.NewHTTP(base)
	http.SetMethod("POST")
	http.SetBody(body)

	if logp.IsDebug(metricsetName) {
		debugf("%v The body for POST requests to jolokia host %v is: %v",
			logPrefix, base.HostData().Host, string(body))
	}

	return &MetricSet{
		BaseMetricSet: base,
		mapping:       mapping,
		namespace:     config.Namespace,
		http:          http,
	}, nil

}

// Fetch methods implements the data gathering and data conversion to the right format
func (m *MetricSet) Fetch() (common.MapStr, error) {
	body, err := m.http.FetchContent()
	if err != nil {
		return nil, err
	}

	if logp.IsDebug(metricsetName) {
		debugf("%v The response body from jolokia host %v is: %v",
			logPrefix, m.HostData().Host, string(body))
	}

	event, err := eventMapping(body, m.mapping)
	if err != nil {
		return nil, err
	}

	// Set dynamic namespace.
	event[mb.NamespaceKey] = m.namespace

	return event, nil
}
