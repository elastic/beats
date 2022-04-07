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

package monitors

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/elastic/beats/v8/heartbeat/eventext"
	"github.com/elastic/beats/v8/heartbeat/look"
	"github.com/elastic/beats/v8/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v8/heartbeat/monitors/wrappers"
	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common"
)

// IPSettings provides common configuration settings for IP resolution and ping
// mode.
type IPSettings struct {
	IPv4 bool     `config:"ipv4"`
	IPv6 bool     `config:"ipv6"`
	Mode PingMode `config:"mode"`
}

// PingMode enumeration for configuring `any` or `all` IPs pinging.
type PingMode uint8

const (
	PingModeUndefined PingMode = iota
	PingAny
	PingAll
)

// DefaultIPSettings provides an instance of default IPSettings to be copied
// when unpacking settings from a common.Config object.
var DefaultIPSettings = IPSettings{
	IPv4: true,
	IPv6: true,
	Mode: PingAny,
}

// emptyTask is a helper value for a Noop.
var emptyTask = MakeSimpleCont(func(*beat.Event) error { return nil })

// Network determines the Network type used for IP pluginName resolution, based on the
// provided settings.
func (s IPSettings) Network() string {
	switch {
	case s.IPv4 && !s.IPv6:
		return "ip4"
	case !s.IPv4 && s.IPv6:
		return "ip6"
	case s.IPv4 && s.IPv6:
		return "ip"
	}
	return ""
}

// MakeSimpleCont wraps a function that produces an event and error
// into an executable Job.
func MakeSimpleCont(f func(*beat.Event) error) jobs.Job {
	return func(event *beat.Event) ([]jobs.Job, error) {
		err := f(event)
		return nil, err
	}
}

// MakePingIPFactory creates a jobFactory for building a Task from a new IP address.
func MakePingIPFactory(
	f func(*beat.Event, *net.IPAddr) error,
) func(*net.IPAddr) jobs.Job {
	return func(ip *net.IPAddr) jobs.Job {
		return MakeSimpleCont(func(event *beat.Event) error {
			return f(event, ip)
		})
	}
}

// MakeByIPJob builds a new Job based on already known IP. Similar to
// MakeByHostJob, the pingFactory will be used to build the tasks run by the job.
//
// A pingFactory instance is normally build with MakePingIPFactory,
// MakePingAllIPFactory or MakePingAllIPPortFactory.
func MakeByIPJob(
	ip net.IP,
	pingFactory func(ip *net.IPAddr) jobs.Job,
) (jobs.Job, error) {
	// use ResolveIPAddr to parse the ip into net.IPAddr adding a zone info
	// if ipv6 is used. We intentionally do not use a custom resolver here.
	addr, err := net.ResolveIPAddr("ip", ip.String())
	if err != nil {
		return nil, err
	}

	fields := common.MapStr{
		"monitor": common.MapStr{"ip": addr.String()},
	}

	return wrappers.WithFields(fields, pingFactory(addr)), nil
}

// MakeByHostJob creates a new Job including host lookup. The pingFactory will be used to
// build one or multiple Tasks after pluginName lookup according to settings.
//
// A pingFactory instance is normally build with MakePingIPFactory,
// MakePingAllIPFactory or MakePingAllIPPortFactory.
func MakeByHostJob(
	host string,
	ipSettings IPSettings,
	resolver Resolver,
	pingFactory func(ip *net.IPAddr) jobs.Job,
) (jobs.Job, error) {
	if ip := net.ParseIP(host); ip != nil {
		return MakeByIPJob(ip, pingFactory)
	}

	network := ipSettings.Network()
	if network == "" {
		return nil, errors.New("pinging hosts requires ipv4 or ipv6 mode enabled")
	}

	mode := ipSettings.Mode

	if mode == PingAny {
		return makeByHostAnyIPJob(host, ipSettings, resolver, pingFactory), nil
	}

	return makeByHostAllIPJob(host, ipSettings, resolver, pingFactory), nil
}

func makeByHostAnyIPJob(
	host string,
	ipSettings IPSettings,
	resolver Resolver,
	pingFactory func(ip *net.IPAddr) jobs.Job,
) jobs.Job {
	network := ipSettings.Network()

	return func(event *beat.Event) ([]jobs.Job, error) {
		resolveStart := time.Now()
		ip, err := resolver.ResolveIPAddr(network, host)
		if err != nil {
			return nil, err
		}

		resolveEnd := time.Now()
		resolveRTT := resolveEnd.Sub(resolveStart)

		ipFields := resolveIPEvent(ip.String(), resolveRTT)
		return wrappers.WithFields(ipFields, pingFactory(ip))(event)
	}
}

func makeByHostAllIPJob(
	host string,
	ipSettings IPSettings,
	resolver Resolver,
	pingFactory func(ip *net.IPAddr) jobs.Job,
) jobs.Job {
	network := ipSettings.Network()
	filter := makeIPFilter(network)

	return func(event *beat.Event) ([]jobs.Job, error) {
		// TODO: check for better DNS IP lookup support:
		//         - The net.LookupIP drops ipv6 zone index
		//
		resolveStart := time.Now()
		ips, err := net.LookupIP(host)
		if err != nil {
			return nil, err
		}

		resolveEnd := time.Now()
		resolveRTT := resolveEnd.Sub(resolveStart)

		if filter != nil {
			ips = filterIPs(ips, filter)
		}

		if len(ips) == 0 {
			err := fmt.Errorf("no %v address resolvable for host %v", network, host)
			return nil, err
		}

		// create ip ping tasks
		cont := make([]jobs.Job, len(ips))
		for i, ip := range ips {
			addr := &net.IPAddr{IP: ip}
			ipFields := resolveIPEvent(ip.String(), resolveRTT)
			cont[i] = wrappers.WithFields(ipFields, pingFactory(addr))
		}
		// Ideally we would test this invocation. This function however is really hard to to test given all the extra context it takes in
		// In a future refactor we could perhaps test that this in correctly invoked.
		eventext.CancelEvent(event)

		return cont, err
	}
}

func resolveIPEvent(ip string, rtt time.Duration) common.MapStr {
	return common.MapStr{
		"monitor": common.MapStr{
			"ip": ip,
		},
		"resolve": common.MapStr{
			"ip":  ip,
			"rtt": look.RTT(rtt),
		},
	}
}

// Unpack sets PingMode from a constant string. Unpack will be called by common.Unpack when
// unpacking into an IPSettings type.
func (p *PingMode) Unpack(s string) error {
	switch s {
	case "all":
		*p = PingAll
	case "any":
		*p = PingAny
	default:
		return fmt.Errorf("expecting 'any' or 'all', not '%v'", s)
	}
	return nil
}

func makeIPFilter(network string) func(net.IP) bool {
	switch network {
	case "ip4":
		return func(i net.IP) bool { return i.To4() != nil }
	case "ip6":
		return func(i net.IP) bool { return i.To4() == nil && i.To16() != nil }
	}
	return nil
}

func filterIPs(ips []net.IP, filt func(net.IP) bool) []net.IP {
	out := ips[:0]
	for _, ip := range ips {
		if filt(ip) {
			out = append(out, ip)
		}
	}
	return out
}
