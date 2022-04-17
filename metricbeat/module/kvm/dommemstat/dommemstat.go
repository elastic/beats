// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package dommemstat

import (
	"net"
	"net/url"
	"time"

	"github.com/menderesk/beats/v7/libbeat/common/cfgwarn"

	"github.com/pkg/errors"

	"github.com/digitalocean/go-libvirt"
	"github.com/digitalocean/go-libvirt/libvirttest"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/metricbeat/mb"
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
	cfgwarn.Beta("The kvm dommemstat metricset is beta.")
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
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
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
			return errors.Wrapf(err, "cannot connect to %v", u)
		}
	}

	defer c.Close()

	l := libvirt.New(c)
	if err = l.Connect(); err != nil {
		return errors.Wrap(err, "error connecting to libvirtd")
	}
	defer func() {
		if err = l.Disconnect(); err != nil {
			msg := errors.Wrap(err, "failed to disconnect")
			report.Error(msg)
			m.Logger().Error(msg)
		}
	}()

	domains, err := l.Domains()
	if err != nil {
		return errors.Wrap(err, "error listing domains")
	}

	for _, d := range domains {
		gotDomainMemoryStats, err := l.DomainMemoryStats(d, maximumStats, flags)
		if err != nil {
			msg := errors.Wrapf(err, "error fetching memory stats for domain %s", d.Name)
			report.Error(msg)
			m.Logger().Error(msg)
			continue
		}

		if len(gotDomainMemoryStats) == 0 {
			msg := errors.Errorf("no memory stats for domain %s", d.Name)
			report.Error(msg)
			m.Logger().Error(msg)
			continue
		}

		for i := range gotDomainMemoryStats {
			reported := report.Event(mb.Event{
				MetricSetFields: common.MapStr{
					"id":   d.ID,
					"name": d.Name,
					"stat": common.MapStr{
						"name":  getDomainMemoryStatName(gotDomainMemoryStats[i].Tag),
						"value": gotDomainMemoryStats[i].Val,
					},
				},
			})
			if !reported {
				return errors.New("metricset has closed")
			}
		}
	}

	return nil
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
		return "actualballoon"
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
