package node

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/munin"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("munin", "node", New,
		mb.DefaultMetricSet(),
	)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	namespace string
	timeout   time.Duration
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Experimental("The munin node metricset is experimental.")

	config := struct {
		Namespace string `config:"node.namespace" validate:"required"`
	}{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		namespace:     config.Namespace,
		timeout:       base.Module().Config().Timeout,
	}, nil
}

// Fetch method implements the data gathering
func (m *MetricSet) Fetch() (common.MapStr, error) {
	node, err := munin.Connect(m.Host(), m.timeout)
	if err != nil {
		return nil, err
	}
	defer node.Close()

	items, err := node.List()
	if err != nil {
		return nil, err
	}

	event, err := node.Fetch(items...)
	if err != nil {
		return nil, err
	}

	// Set dynamic namespace.
	_, err = event.Put(mb.NamespaceKey, m.namespace)
	if err != nil {
		return nil, err
	}

	return event, nil

}
