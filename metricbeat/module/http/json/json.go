package json

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	if err := mb.Registry.AddMetricSet("http", "json", New, hostParser); err != nil {
		panic(err)
	}
}

const (
	// defaultScheme is the default scheme to use when it is not specified in the host config.
	defaultScheme = "http"

	// defaultPath is the dto use when it is not specified in the host config.
	defaultPath = ""
)

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		PathConfigKey: "path",
		DefaultPath:   defaultPath,
	}.Build()
)

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	mb.BaseMetricSet
	namespace       string
	http            *helper.HTTP
	method          string
	body            string
	requestEnabled  bool
	responseEnabled bool
	jsonIsArray     bool
	deDotEnabled    bool
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	config := struct {
		Namespace       string `config:"namespace" validate:"required"`
		Method          string `config:"method"`
		Body            string `config:"body"`
		RequestEnabled  bool   `config:"request.enabled"`
		ResponseEnabled bool   `config:"response.enabled"`
		JSONIsArray     bool   `config:"json.is_array"`
		DeDotEnabled    bool   `config:"dedot.enabled"`
	}{
		Method:          "GET",
		Body:            "",
		RequestEnabled:  false,
		ResponseEnabled: false,
		JSONIsArray:     false,
		DeDotEnabled:    false,
	}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	http, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}
	http.SetMethod(config.Method)
	http.SetBody([]byte(config.Body))

	return &MetricSet{
		BaseMetricSet:   base,
		namespace:       config.Namespace,
		method:          config.Method,
		body:            config.Body,
		http:            http,
		requestEnabled:  config.RequestEnabled,
		responseEnabled: config.ResponseEnabled,
		jsonIsArray:     config.JSONIsArray,
		deDotEnabled:    config.DeDotEnabled,
	}, nil
}

func (m *MetricSet) processBody(response *http.Response, jsonBody interface{}) common.MapStr {
	var event common.MapStr

	if m.deDotEnabled {
		event = common.DeDotJSON(jsonBody).(common.MapStr)
	} else {
		event = jsonBody.(common.MapStr)
	}

	if m.requestEnabled {
		event[mb.ModuleDataKey] = common.MapStr{
			"request": common.MapStr{
				"headers": m.getHeaders(response.Request.Header),
				"method":  response.Request.Method,
				"body":    m.body,
			},
		}
	}

	if m.responseEnabled {
		phrase := strings.TrimPrefix(response.Status, strconv.Itoa(response.StatusCode)+" ")
		event[mb.ModuleDataKey] = common.MapStr{
			"response": common.MapStr{
				"code":    response.StatusCode,
				"phrase":  phrase,
				"headers": m.getHeaders(response.Header),
			},
		}
	}

	// Set dynamic namespace
	event["_namespace"] = m.namespace

	return event
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	response, err := m.http.FetchResponse()
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	var jsonBody common.MapStr
	var jsonBodyArr []common.MapStr
	var events []common.MapStr

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if m.jsonIsArray {
		err = json.Unmarshal(body, &jsonBodyArr)
		if err != nil {
			return nil, err
		}

		for _, obj := range jsonBodyArr {
			event := m.processBody(response, obj)
			events = append(events, event)
		}
	} else {
		err = json.Unmarshal(body, &jsonBody)
		if err != nil {
			return nil, err
		}

		event := m.processBody(response, jsonBody)
		events = append(events, event)
	}

	return events, nil
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
