package json

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/mb"
	"io/ioutil"
	"net/http"
	"strings"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	if err := mb.Registry.AddMetricSet("http", "json", New); err != nil {
		panic(err)
	}
}

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	mb.BaseMetricSet
	http    *helper.HTTP
	headers map[string]string
	method  string
	body    string
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	config := struct {
		Method  string            `config:"method"`
		Body    string            `config:"body"`
		Headers map[string]string `config:"headers"`
	}{}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	http := helper.NewHTTP(base)
	http.SetMethod(config.Method)
	http.SetBody([]byte(config.Body))
	for key, value := range config.Headers {
		http.SetHeader(key, value)
	}

	return &MetricSet{
		BaseMetricSet: base,
		http:          http,
		headers:       config.Headers,
		method:        config.Method,
		body:          config.Body,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch() (common.MapStr, error) {

	response, err := m.http.FetchResponse()
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	event := common.MapStr{}

	event["request"] = common.MapStr{
		"headers": m.headers,
		"method":  m.method,
		"body":    m.body,
	}

	event["response"] = common.MapStr{
		"status_code": response.StatusCode,
		"headers":     m.getHeaders(response.Header),
		"body":        responseBody,
	}

	return event, nil
}

func (m *MetricSet) getHeaders(header http.Header) map[string]string {

	headers := make(map[string]string)
	for k, v := range header {
		value := ""
		for _, h := range v {
			value += h + " ,"
		}
		value = strings.TrimRight(value, " ,")
		headers[k] = value
	}
	return headers
}
