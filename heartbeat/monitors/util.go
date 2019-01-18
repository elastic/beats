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

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"

	"github.com/elastic/beats/heartbeat/look"
)

// IPSettings provides common configuration settings for IP resolution and ping
// mode.
type IPSettings struct {
	IPv4 bool     `config:"ipv4"`
	IPv6 bool     `config:"ipv6"`
	Mode PingMode `config:"mode"`
}

// HostJobSettings configures a Job including Host lookups and global fields to be added
// to every event.
type HostJobSettings struct {
	Name   string
	Host   string
	IP     IPSettings
	Fields common.MapStr
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

// Network determines the Network type used for IP name resolution, based on the
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

// WithErrAsField wraps the given Job's execution such that any error returned
// by the original Job will be set as a field. The original error will not be
// passed through as a return value. Errors may still be present but only if there
// is an actual error wrapping the error.
func WithErrAsField(job Job) Job {
	return AfterJob(job, func(event *beat.Event, jobs []Job, err error) ([]Job, error) {
		if err != nil {
			// Handle the case where we have a parent configuredJob that only spawns subtasks
			// that has itself encountered an error
			if event == nil {
				event = &beat.Event{}
			}
			MergeEventFields(event, common.MapStr{
				"error": look.Reason(err),
			})
		}

		wrapped := WrapAll(jobs, WithErrAsField)
		return wrapped, nil
	})
}

// TimeAndCheckJob executes the given Job, checking the duration of its run and setting
// its status.
// It adds the monitor.duration and monitor.status fields.
func TimeAndCheckJob(job Job) Job {
	return CreateNamedJob(
		job.Name(),
		func(event *beat.Event) ([]Job, error) {
			start := time.Now()

			cont, err := job.Run(event)

			if event != nil {
				status := look.Status(err)
				MergeEventFields(event, common.MapStr{
					"monitor": common.MapStr{
						"duration": look.RTT(time.Since(start)),
						"status":   status,
					},
				})
				event.Timestamp = start
			}

			wrappedCont := WrapAll(cont, TimeAndCheckJob)
			return wrappedCont, err
		},
	)
}

// WithJobId wraps the given Job setting the monitor.id field.
func WithJobId(id string, job Job) Job {
	return CreateNamedJob(
		id,
		WithFields(
			common.MapStr{
				"monitor": common.MapStr{"id": id},
			},
			job,
		).Run,
	)
}

// MakeSimpleCont wraps a function that produces an event and error
// into an executable Job.
func MakeSimpleCont(f func(*beat.Event) error) Job {
	return AnonJob(func(event *beat.Event) ([]Job, error) {
		err := f(event)
		return nil, err
	})
}

// MakePingIPFactory creates a jobFactory for building a Task from a new IP address.
func MakePingIPFactory(
	f func(*beat.Event, *net.IPAddr) error,
) func(*net.IPAddr) Job {
	return func(ip *net.IPAddr) Job {
		return MakeSimpleCont(func(event *beat.Event) error {
			return f(event, ip)
		})
	}
}

// MakePingAllIPFactory wraps a function for building a recursive Task Runner from function callbacks.
func MakePingAllIPFactory(
	f func(*net.IPAddr) []func(*beat.Event) error,
) func(*net.IPAddr) Job {
	return func(ip *net.IPAddr) Job {
		cont := f(ip)
		switch len(cont) {
		case 0:
			return emptyTask
		case 1:
			return MakeSimpleCont(cont[0])
		}

		tasks := make([]Job, len(cont))
		for i, c := range cont {
			tasks[i] = MakeSimpleCont(c)
		}
		return AnonJob(func(event *beat.Event) ([]Job, error) {
			return tasks, nil
		})
	}
}

// MakePingAllIPPortFactory builds a set of TaskRunner supporting a set of
// IP/port-pairs.
func MakePingAllIPPortFactory(
	ports []uint16,
	f func(*beat.Event, *net.IPAddr, uint16) error,
) func(*net.IPAddr) Job {
	if len(ports) == 1 {
		port := ports[0]
		return MakePingIPFactory(func(event *beat.Event, ip *net.IPAddr) error {
			return f(event, ip, port)
		})
	}

	return MakePingAllIPFactory(func(ip *net.IPAddr) []func(event *beat.Event) error {
		funcs := make([]func(*beat.Event) error, len(ports))
		for i := range ports {
			port := ports[i]
			funcs[i] = func(event *beat.Event) error {
				return f(event, ip, port)
			}
		}
		return funcs
	})
}

// MakeByIPJob builds a new Job based on already known IP. Similar to
// MakeByHostJob, the pingFactory will be used to build the tasks run by the job.
//
// A pingFactory instance is normally build with MakePingIPFactory,
// MakePingAllIPFactory or MakePingAllIPPortFactory.
func MakeByIPJob(
	ip net.IP,
	pingFactory func(ip *net.IPAddr) Job,
) (Job, error) {
	// use ResolveIPAddr to parse the ip into net.IPAddr adding a zone info
	// if ipv6 is used.
	addr, err := net.ResolveIPAddr("ip", ip.String())
	if err != nil {
		return nil, err
	}

	fields := common.MapStr{
		"monitor": common.MapStr{"ip": addr.String()},
	}

	return TimeAndCheckJob(WithFields(fields, pingFactory(addr))), nil
}

// MakeByHostJob creates a new Job including host lookup. The pingFactory will be used to
// build one or multiple Tasks after name lookup according to settings.
//
// A pingFactory instance is normally build with MakePingIPFactory,
// MakePingAllIPFactory or MakePingAllIPPortFactory.
func MakeByHostJob(
	settings HostJobSettings,
	pingFactory func(ip *net.IPAddr) Job,
) (Job, error) {
	host := settings.Host

	if ip := net.ParseIP(host); ip != nil {
		return MakeByIPJob(ip, pingFactory)
	}

	network := settings.IP.Network()
	if network == "" {
		return nil, errors.New("pinging hosts requires ipv4 or ipv6 mode enabled")
	}

	mode := settings.IP.Mode

	if mode == PingAny {
		return makeByHostAnyIPJob(settings, host, pingFactory), nil
	}

	return TimeAndCheckJob(makeByHostAllIPJob(settings, host, pingFactory)), nil
}

func makeByHostAnyIPJob(
	settings HostJobSettings,
	host string,
	pingFactory func(ip *net.IPAddr) Job,
) Job {
	network := settings.IP.Network()

	aj := AnonJob(func(event *beat.Event) ([]Job, error) {
		resolveStart := time.Now()
		ip, err := net.ResolveIPAddr(network, host)
		if err != nil {
			resolveErr(event, host, err)
			return nil, nil
		}

		resolveEnd := time.Now()
		resolveRTT := resolveEnd.Sub(resolveStart)

		ipFields := resolveIPEvent(host, ip.String(), resolveRTT)
		return WithFields(ipFields, pingFactory(ip)).Run(event)
	})

	return TimeAndCheckJob(aj)
}

func makeByHostAllIPJob(
	settings HostJobSettings,
	host string,
	pingFactory func(ip *net.IPAddr) Job,
) Job {
	network := settings.IP.Network()
	filter := makeIPFilter(network)

	return CreateNamedJob(settings.Name, func(event *beat.Event) ([]Job, error) {
		// TODO: check for better DNS IP lookup support:
		//         - The net.LookupIP drops ipv6 zone index
		//
		resolveStart := time.Now()
		ips, err := net.LookupIP(host)
		if err != nil {
			resolveErr(event, host, err)
			return nil, nil
		}

		resolveEnd := time.Now()
		resolveRTT := resolveEnd.Sub(resolveStart)

		if filter != nil {
			ips = filterIPs(ips, filter)
		}

		if len(ips) == 0 {
			err := fmt.Errorf("no %v address resolvable for host %v", network, host)
			resolveErr(event, host, err)
			return nil, nil
		}

		// create ip ping tasks
		cont := make([]Job, len(ips))
		for i, ip := range ips {
			addr := &net.IPAddr{IP: ip}
			ipFields := resolveIPEvent(host, ip.String(), resolveRTT)
			cont[i] = TimeAndCheckJob(WithFields(ipFields, pingFactory(addr)))
		}
		return cont, nil
	})
}

func resolveIPEvent(host, ip string, rtt time.Duration) common.MapStr {
	return common.MapStr{
		"monitor": common.MapStr{
			"host": host,
			"ip":   ip,
		},
		"resolve": common.MapStr{
			"host": host,
			"ip":   ip,
			"rtt":  look.RTT(rtt),
		},
	}
}

func resolveErr(event *beat.Event, host string, err error) {
	MergeEventFields(event, common.MapStr{
		"monitor": common.MapStr{
			"host": host,
		},
		"resolve": common.MapStr{
			"host": host,
		},
	})
}

// WithFields wraps a TaskRunner, updating all events returned with the set of
// fields configured.
func WithFields(fields common.MapStr, job Job) Job {
	return AfterJob(job, func(event *beat.Event, cont []Job, err error) ([]Job, error) {
		MergeEventFields(event, fields)

		for i := range cont {
			cont[i] = WithFields(fields, cont[i])
		}
		return cont, err
	})
}

func withStart(field string, start time.Time, r Job) Job {
	return AfterJobSuccess(r, func(event *beat.Event, cont []Job, err error) ([]Job, error) {
		event.Fields.Put(field, look.RTT(time.Since(start)))

		for i := range cont {
			cont[i] = withStart(field, start, cont[i])
		}

		return cont, err
	})
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

// MakeHostJobSettings creates a new HostJobSettings structure without any global
// event fields.
func MakeHostJobSettings(name, host string, ip IPSettings) HostJobSettings {
	return HostJobSettings{Name: name, Host: host, IP: ip}
}

// WithFields adds new event fields to a Job. Existing fields will be
// overwritten.
// The fields map will be updated (no copy).
func (s HostJobSettings) WithFields(m common.MapStr) HostJobSettings {
	s.AddFields(m)
	return s
}

// AddFields adds new event fields to a Job. Existing fields will be
// overwritten.
func (s *HostJobSettings) AddFields(m common.MapStr) { addFields(&s.Fields, m) }

func addFields(to *common.MapStr, m common.MapStr) {
	if m == nil {
		return
	}

	fields := *to
	if fields == nil {
		fields = common.MapStr{}
		*to = fields
	}
	fields.DeepUpdate(m)
}

// MergeEventFields merges the given common.MapStr into the given Event's Fields.
func MergeEventFields(e *beat.Event, merge common.MapStr) {
	if e.Fields != nil {
		e.Fields.DeepUpdate(merge)
	} else {
		e.Fields = merge
	}
}
