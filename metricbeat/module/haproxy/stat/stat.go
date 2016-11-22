package stat

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/haproxy"

	"github.com/pkg/errors"
)

const (
	statsMethod = "stat"
)

var (
	debugf = logp.MakeDebug("haproxy-stat")
)

// init registers the haproxy stat MetricSet.
func init() {
	if err := mb.Registry.AddMetricSet("haproxy", statsMethod, New, haproxy.HostParser); err != nil {
		panic(err)
	}
}

// MetricSet for haproxy stats.
type MetricSet struct {
	mb.BaseMetricSet
}

// New creates a new haproxy stat MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	logp.Warn("EXPERIMENTAL: The %v %v metricset is experimental", base.Module().Name(), base.Name())

	return &MetricSet{BaseMetricSet: base}, nil
}

// Fetch methods returns a list of stats metrics.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	// haproxy doesn't accept a username or password so ignore them if they
	// are in the URL.
	hapc, err := haproxy.NewHaproxyClient(m.HostData().SanitizedURI)
	if err != nil {
		return nil, errors.Wrap(err, "failed creating haproxy client")
	}

	res, err := hapc.GetStat()
	if err != nil {
		return nil, errors.Wrap(err, "failed fetching haproxy stat")
	}

	return eventMapping(res), nil
}
