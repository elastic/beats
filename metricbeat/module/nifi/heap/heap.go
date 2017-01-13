package heap

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	if err := mb.Registry.AddMetricSet("nifi", "heap", New); err != nil {
		panic(err)
	}
}

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	mb.BaseMetricSet
	client *http.Client
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	logp.Warn("EXPERIMENTAL: The %v %v metricset is experimental", base.Module().Name(), base.Name())

	config := struct{}{}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		client:        &http.Client{Timeout: base.Module().Config().Timeout},
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch() (common.MapStr, error) {

	fmt.Println("LN 54")
	url := fmt.Sprintf("http://%s/nifi-api/system-diagnostics", m.HostData().URI)

	fmt.Println(url)

	req, _ := http.NewRequest("GET", url, nil)

	resp, err := m.client.Do(req)
	if err != nil {
		msg := fmt.Sprintf("Error making HTTP request: %v", err)
		fmt.Println(msg)
		logp.Err(msg)
		return nil, errors.New(msg)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		msg := fmt.Sprintf("Non-200 Response returned from NiFi request: %d: %s", resp.StatusCode, resp.Status)
		fmt.Println(msg)
		logp.Err(msg)
		return nil, errors.New(msg)
	}

	event := eventMapping(resp.Body)
	fmt.Printf("%v", event)
	return event, nil
}
