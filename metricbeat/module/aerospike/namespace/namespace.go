package namespace

import (
	"strings"

	as "github.com/aerospike/aerospike-client-go"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/aerospike"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	mb.Registry.MustAddMetricSet("aerospike", "namespace", New,
		mb.DefaultMetricSet(),
	)
}

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	mb.BaseMetricSet
	host   *as.Host
	client *as.Client
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := struct{}{}

	cfgwarn.Beta("The aerospike namespace metricset is beta")

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	host, err := aerospike.ParseHost(base.Host())
	if err != nil {
		return nil, errors.Wrap(err, "Invalid host format, expected hostname:port")
	}

	return &MetricSet{
		BaseMetricSet: base,
		host:          host,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	var events []common.MapStr

	if err := m.connect(); err != nil {
		return nil, err
	}

	for _, node := range m.client.GetNodes() {
		info, err := as.RequestNodeInfo(node, "namespaces")
		if err != nil {
			logp.Err("Failed to retrieve namespaces from node %s", node.GetName())
			continue
		}

		for _, namespace := range strings.Split(info["namespaces"], ";") {
			info, err := as.RequestNodeInfo(node, "namespace/"+namespace)
			if err != nil {
				logp.Err("Failed to retrieve metrics for namespace %s from node %s", namespace, node.GetName())
				continue
			}

			data, _ := schema.Apply(aerospike.ParseInfo(info["namespace/"+namespace]))
			data["name"] = namespace
			data["node"] = common.MapStr{
				"host": node.GetHost().String(),
				"name": node.GetName(),
			}

			events = append(events, data)
		}
	}

	return events, nil
}

// create an aerospike client if it doesn't exist yet
func (m *MetricSet) connect() error {
	if m.client == nil {
		client, err := as.NewClientWithPolicyAndHost(as.NewClientPolicy(), m.host)
		if err != nil {
			return err
		}
		m.client = client
	}
	return nil
}
