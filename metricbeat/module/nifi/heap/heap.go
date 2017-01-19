package heap

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"

	"github.com/elastic/beats/metricbeat/module/nifi"
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
	client    *http.Client
	nodes     map[string]string
	isCluster bool
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

	arbHost := base.Module().Config().Hosts[0]

	client := &http.Client{Timeout: base.Module().Config().Timeout}

	isCluster := nifi.IsCluster(arbHost, client)

	var nodes map[string]string

	if isCluster {
		nodesTmp, err := nifi.GetNodeMap(arbHost, client)
		if err != nil {
			logp.Err(err.Error())
			return nil, err
		}
		nodes = nodesTmp
	}

	return &MetricSet{
		BaseMetricSet: base,
		client:        client,
		nodes:         nodes,
		isCluster:     isCluster,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch() (common.MapStr, error) {

	var event common.MapStr

	if m.isCluster {
		eventTmp, err := m.fetchNodewise()
		if err != nil {
			logp.Err(err.Error())
			return nil, err
		}
		event = eventTmp

	} else {
		eventTmp, err := m.fetchAggregate()
		if err != nil {
			logp.Err(err.Error())
			return nil, err
		}
		event = eventTmp
	}

	return event, nil
}

func (m *MetricSet) fetchNodewise() (common.MapStr, error) {
	host := m.HostData().URI
	nodeID := m.nodes[host]

	url := fmt.Sprintf("http://%s/nifi-api/system-diagnostics?clusterNodeId=%s", host, nodeID)

	req, _ := http.NewRequest("GET", url, nil)

	resp, err := m.client.Do(req)
	if err != nil {
		msg := fmt.Sprintf("Error making HTTP request: %v", err)
		logp.Err(msg)
		return nil, errors.New(msg)
	}

	if resp.StatusCode != 200 {
		msg := fmt.Sprintf("Non-200 Response returned from NiFi request: %d: %s", resp.StatusCode, resp.Status)
		logp.Err(msg)
		return nil, errors.New(msg)
	}

	event, err := nodewiseEventMapping(resp.Body, nodeID)
	if err != nil {
		logp.Err(err.Error())
		return nil, err
	}
	fmt.Printf("%v", event)

	return event, nil

}

func (m *MetricSet) fetchAggregate() (common.MapStr, error) {
	url := fmt.Sprintf("http://%s/nifi-api/system-diagnostics", m.HostData().URI)
	fmt.Println(url)

	req, _ := http.NewRequest("GET", url, nil)

	resp, err := m.client.Do(req)
	if err != nil {
		msg := fmt.Sprintf("Error making HTTP request: %v", err)
		logp.Err(msg)
		return nil, errors.New(msg)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		msg := fmt.Sprintf("Non-200 Response returned from NiFi request: %d: %s", resp.StatusCode, resp.Status)
		logp.Err(msg)
		return nil, errors.New(msg)
	}

	event := eventMapping(resp.Body)
	fmt.Printf("%v", event)

	return event, nil
}
