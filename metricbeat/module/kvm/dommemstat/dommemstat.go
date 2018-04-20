package dommemstat

import (
	"errors"
	"net"
	"net/url"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"

	"github.com/digitalocean/go-libvirt"
	"github.com/digitalocean/go-libvirt/libvirttest"
)

const (
	// maximum number of memory stats to be collected
	// limit is defined by REMOTE_DOMAIN_MEMORY_STATS_MAX
	// based on https://github.com/libvirt/libvirt/blob/5bb07527c11a6123e044a5dfc48bdeccee144994/src/remote/remote_protocol.x#L136
	maximumStats = 11
	// flag VIR_DOMAIN_AFFECT_CURRENT passed to collect memory stats
	// based on https://libvirt.org/html/libvirt-libvirt-domain.html#virDomainModificationImpact
	flags = 0
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("kvm", "dommemstat", New,
		mb.DefaultMetricSet(),
	)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	Timeout time.Duration
	HostURL *url.URL
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Experimental("The kvm dommemstat metricset is experimental.")

	u, err := url.Parse(base.HostData().URI)
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		Timeout:       base.Module().Config().Timeout,
		HostURL:       u,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) {

	var (
		c   net.Conn
		err error
	)

	u := m.HostURL

	if u.Scheme == "test" {
		// when running tests, a mock Libvirt server is used
		c = libvirttest.New()
	} else {
		address := u.Host
		if u.Host == "" {
			address = u.Path
		}

		c, err = net.DialTimeout(u.Scheme, address, m.Timeout)
		if err != nil {
			report.Error(err)
		}
	}

	defer c.Close()

	l := libvirt.New(c)
	if err := l.Connect(); err != nil {
		report.Error(err)
	}

	domains, err := l.Domains()
	if err != nil {
		report.Error(err)
	}

	for _, d := range domains {
		gotDomainMemoryStats, err := l.DomainMemoryStats(d, maximumStats, flags)
		if err != nil {
			report.Error(err)
		}

		if len(gotDomainMemoryStats) == 0 {
			report.Error(errors.New("no domain memory stats found"))
		}

		for i := range gotDomainMemoryStats {
			report.Event(mb.Event{
				MetricSetFields: common.MapStr{
					"id":   d.ID,
					"name": d.Name,
					"stat": common.MapStr{
						"name":  getDomainMemoryStatName(gotDomainMemoryStats[i].Tag),
						"value": gotDomainMemoryStats[i].Val,
					},
				},
			})
		}
	}

	if err := l.Disconnect(); err != nil {
		report.Error(errors.New("failed to disconnect"))
	}
}

func getDomainMemoryStatName(tag int32) string {
	// this is based on https://github.com/digitalocean/go-libvirt/blob/59d541f19311883ad82708651353009fb207d8a9/const.gen.go#L718
	switch tag {
	case 0:
		return "swapin"
	case 1:
		return "swapout"
	case 2:
		return "majorfault"
	case 3:
		return "minorfault"
	case 4:
		return "unused"
	case 5:
		return "available"
	case 6:
		return "actualballon"
	case 7:
		return "rss"
	case 8:
		return "usable"
	case 9:
		return "lastupdate"
	case 10:
		return "nr"
	default:
		return "unidentified"
	}
}
