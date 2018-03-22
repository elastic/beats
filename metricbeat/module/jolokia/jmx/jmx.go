package jmx

import (
	"github.com/joeshaw/multierror"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

var (
	metricsetName = "jolokia.jmx"
)

// init registers the MetricSet with the central registry.
func init() {
	if err := mb.Registry.AddMetricSet("jolokia", "jmx", New, hostParser); err != nil {
		panic(err)
	}
}

const (
	defaultScheme = "http"
	defaultPath   = "/jolokia/?ignoreErrors=true&canonicalNaming=false"
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
	mapping   AttributeMapping
	namespace string
	http      *helper.HTTP
	log       *logp.Logger
}

// New create a new instance of the MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

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

	http, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}
	http.SetMethod("POST")
	http.SetBody(body)

	log := logp.NewLogger(metricsetName).With("host", base.HostData().Host)

	if logp.IsDebug(metricsetName) {
		log.Debugw("Jolokia request body",
			"body", string(body), "type", "request")
	}

	return &MetricSet{
		BaseMetricSet: base,
		mapping:       mapping,
		namespace:     config.Namespace,
		http:          http,
		log:           log,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	body, err := m.http.FetchContent()
	if err != nil {
		return nil, err
	}

	if logp.IsDebug(metricsetName) {
		m.log.Debugw("Jolokia response body",
			"host", m.HostData().Host, "body", string(body), "type", "response")
	}

	events, err := eventMapping(body, m.mapping)
	if err != nil {
		return nil, err
	}

	// Set dynamic namespace.
	var errs multierror.Errors
	for _, event := range events {
		_, err = event.Put(mb.NamespaceKey, m.namespace)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return events, errs.Err()
}
