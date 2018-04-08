package info

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/haproxy"

	"github.com/pkg/errors"
)

const (
	statsMethod = "info"
)

var (
	debugf = logp.MakeDebug("haproxy-info")
)

// init registers the haproxy info MetricSet.
func init() {
	mb.Registry.MustAddMetricSet("haproxy", "info", New,
		mb.WithHostParser(haproxy.HostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet for haproxy info.
type MetricSet struct {
	mb.BaseMetricSet
}

// New creates a haproxy info MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return &MetricSet{BaseMetricSet: base}, nil
}

// Fetch fetches info stats from the haproxy service.
func (m *MetricSet) Fetch() (common.MapStr, error) {
	hapc, err := haproxy.NewHaproxyClient(m.HostData().URI)
	if err != nil {
		return nil, errors.Wrap(err, "failed creating haproxy client")
	}

	res, err := hapc.GetInfo()
	if err != nil {
		return nil, errors.Wrap(err, "failed fetching haproxy info")
	}

	return eventMapping(res)
}
