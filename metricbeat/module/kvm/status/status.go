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

package status

import (
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/digitalocean/go-libvirt"
	"github.com/digitalocean/go-libvirt/libvirttest"
	"github.com/digitalocean/go-libvirt/socket"
	"github.com/digitalocean/go-libvirt/socket/dialers"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("kvm", "status", New,
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
	cfgwarn.Beta("The kvm status metricset is beta.")
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
		d   socket.Dialer
		err error
	)

	u := m.HostURL

	if u.Scheme == "test" {
		// when running tests, a mock Libvirt server is used
		d = libvirttest.New()
	} else {
		address := u.Host
		if u.Host == "" {
			address = u.Path
		}

		c, err := net.DialTimeout(u.Scheme, address, m.Timeout)
		if err != nil {
			return fmt.Errorf("cannot connect to %v: %w", u, err)
		}

		d = dialers.NewAlreadyConnected(c)
		defer c.Close()
	}

	l := libvirt.NewWithDialer(d)
	if err = l.Connect(); err != nil {
		return fmt.Errorf("error connecting to libvirtd: %w", err)
	}
	defer func() {
		if err = l.Disconnect(); err != nil {
			msg := fmt.Errorf("failed to disconnect: %w", err)
			report.Error(msg)
			m.Logger().Error(msg)
		}
	}()

	domains, _, err := l.ConnectListAllDomains(1, libvirt.ConnectListDomainsActive|libvirt.ConnectListDomainsInactive)
	if err != nil {
		return fmt.Errorf("error listing domains: %w", err)
	}

	for _, d := range domains {
		d, err := l.DomainLookupByName(d.Name)
		if err != nil {
			continue
		}

		state, _, err := l.DomainGetState(d, 0)
		if err != nil {
			continue
		}
		reported := report.Event(mb.Event{
			ModuleFields: mapstr.M{
				"id":   d.ID,
				"name": d.Name,
			},
			MetricSetFields: mapstr.M{
				"state": getDomainStateName(libvirt.DomainState(state)),
			},
		})
		if !reported {
			return nil
		}
	}

	return nil
}

func getDomainStateName(tag libvirt.DomainState) string {
	switch tag {
	case 0:
		return "no state"
	case 1:
		return "running"
	case 2:
		return "blocked"
	case 3:
		return "paused"
	case 4:
		return "shutdown"
	case 5:
		return "shutoff"
	case 6:
		return "crashed"
	case 7:
		return "suspended"
	default:
		return "unidentified"
	}
}
