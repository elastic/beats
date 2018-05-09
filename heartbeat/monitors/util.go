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

type funcJob struct {
	settings JobSettings
	run      JobRunner
}

type funcTask struct {
	run func() (common.MapStr, []TaskRunner, error)
}

// IPSettings provides common configuration settings for IP resolution and ping
// mode.
type IPSettings struct {
	IPv4 bool     `config:"ipv4"`
	IPv6 bool     `config:"ipv6"`
	Mode PingMode `config:"mode"`
}

// JobSettings configures a Job name and global fields to be added to every
// event.
type JobSettings struct {
	Name   string
	Fields common.MapStr
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

// MakeSimpleJob creates a new Job from a callback function. The callback should
// return an valid event and can not create any sub-tasks to be executed after
// completion.
func MakeSimpleJob(settings JobSettings, f func() (common.MapStr, error)) Job {
	return MakeJob(settings, func() (common.MapStr, []TaskRunner, error) {
		event, err := f()
		return event, nil, err
	})
}

// MakeJob create a new Job from a callback function. The callback can
// optionally return an event to be published and a set of derived sub-tasks to be
// scheduled. The sub-tasks will be run only once and removed from the scheduler
// after completion.
func MakeJob(settings JobSettings, f func() (common.MapStr, []TaskRunner, error)) Job {
	settings.AddFields(common.MapStr{
		"monitor": common.MapStr{
			"id": settings.Name,
		},
	})

	return &funcJob{settings, func() (beat.Event, []JobRunner, error) {
		// Create and run new annotated Job whenever the Jobs root is Task is executed.
		// This will set the jobs active start timestamp to the time.Now().
		return annotated(settings, time.Now(), f)()
	}}
}

// annotated lifts a TaskRunner into a job, annotating events with common fields and start timestamp.
func annotated(
	settings JobSettings,
	start time.Time,
	fn func() (common.MapStr, []TaskRunner, error),
) JobRunner {
	return func() (beat.Event, []JobRunner, error) {
		var event beat.Event

		fields, cont, err := fn()
		if err != nil {
			if fields == nil {
				fields = common.MapStr{}
			}
			fields["error"] = look.Reason(err)
		}

		if fields != nil {
			fields = fields.Clone()

			status := look.Status(err)
			fields.DeepUpdate(common.MapStr{
				"monitor": common.MapStr{
					"duration": look.RTT(time.Since(start)),
					"status":   status,
				},
			})
			if user := settings.Fields; user != nil {
				fields.DeepUpdate(user.Clone())
			}

			event.Timestamp = start
			event.Fields = fields
		}

		jobCont := make([]JobRunner, len(cont))
		for i, c := range cont {
			jobCont[i] = annotated(settings, start, c.Run)
		}
		return event, jobCont, nil
	}
}

// MakeCont wraps a function into an executable TaskRunner. The task being generated
// can optionally return an event and/or sub-tasks.
func MakeCont(f func() (common.MapStr, []TaskRunner, error)) TaskRunner {
	return funcTask{f}
}

// MakeSimpleCont wraps a function into an executable TaskRunner. The task bein generated
// should return an event to be reported.
func MakeSimpleCont(f func() (common.MapStr, error)) TaskRunner {
	return MakeCont(func() (common.MapStr, []TaskRunner, error) {
		event, err := f()
		return event, nil, err
	})
}

// MakePingIPFactory creates a factory for building a Task from a new IP address.
func MakePingIPFactory(
	f func(*net.IPAddr) (common.MapStr, error),
) func(*net.IPAddr) TaskRunner {
	return func(ip *net.IPAddr) TaskRunner {
		return MakeSimpleCont(func() (common.MapStr, error) { return f(ip) })
	}
}

var emptyTask = MakeSimpleCont(func() (common.MapStr, error) { return nil, nil })

// MakePingAllIPFactory wraps a function for building a recursive Task Runner from function callbacks.
func MakePingAllIPFactory(
	f func(*net.IPAddr) []func() (common.MapStr, error),
) func(*net.IPAddr) TaskRunner {
	return func(ip *net.IPAddr) TaskRunner {
		cont := f(ip)
		switch len(cont) {
		case 0:
			return emptyTask
		case 1:
			return MakeSimpleCont(cont[0])
		}

		tasks := make([]TaskRunner, len(cont))
		for i, c := range cont {
			tasks[i] = MakeSimpleCont(c)
		}
		return MakeCont(func() (common.MapStr, []TaskRunner, error) {
			return nil, tasks, nil
		})
	}
}

// MakePingAllIPPortFactory builds a set of TaskRunner supporting a set of
// IP/port-pairs.
func MakePingAllIPPortFactory(
	ports []uint16,
	f func(*net.IPAddr, uint16) (common.MapStr, error),
) func(*net.IPAddr) TaskRunner {
	if len(ports) == 1 {
		port := ports[0]
		return MakePingIPFactory(func(ip *net.IPAddr) (common.MapStr, error) {
			return f(ip, port)
		})
	}

	return MakePingAllIPFactory(func(ip *net.IPAddr) []func() (common.MapStr, error) {
		funcs := make([]func() (common.MapStr, error), len(ports))
		for i := range ports {
			port := ports[i]
			funcs[i] = func() (common.MapStr, error) {
				return f(ip, port)
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
	settings JobSettings,
	ip net.IP,
	pingFactory func(ip *net.IPAddr) TaskRunner,
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
	return MakeJob(settings, WithFields(fields, pingFactory(addr)).Run), nil
}

// MakeByHostJob creates a new Job including host lookup. The pingFactory will be used to
// build one or multiple Tasks after name lookup according to settings.
//
// A pingFactory instance is normally build with MakePingIPFactory,
// MakePingAllIPFactory or MakePingAllIPPortFactory.
func MakeByHostJob(
	settings HostJobSettings,
	pingFactory func(ip *net.IPAddr) TaskRunner,
) (Job, error) {
	host := settings.Host

	if ip := net.ParseIP(host); ip != nil {
		return MakeByIPJob(settings.jobSettings(), ip, pingFactory)
	}

	network := settings.IP.Network()
	if network == "" {
		return nil, errors.New("pinging hosts requires ipv4 or ipv6 mode enabled")
	}

	mode := settings.IP.Mode

	settings.AddFields(common.MapStr{
		"monitor": common.MapStr{
			"host": host,
		},
	})

	if mode == PingAny {
		return makeByHostAnyIPJob(settings, host, pingFactory), nil
	}
	return makeByHostAllIPJob(settings, host, pingFactory), nil
}

func makeByHostAnyIPJob(
	settings HostJobSettings,
	host string,
	pingFactory func(ip *net.IPAddr) TaskRunner,
) Job {
	network := settings.IP.Network()

	return MakeJob(settings.jobSettings(), func() (common.MapStr, []TaskRunner, error) {
		resolveStart := time.Now()
		ip, err := net.ResolveIPAddr(network, host)
		if err != nil {
			return resolveErr(host, err)
		}

		resolveEnd := time.Now()
		resolveRTT := resolveEnd.Sub(resolveStart)

		event := resolveIPEvent(host, ip.String(), resolveRTT)
		return WithFields(event, pingFactory(ip)).Run()
	})
}

func makeByHostAllIPJob(
	settings HostJobSettings,
	host string,
	pingFactory func(ip *net.IPAddr) TaskRunner,
) Job {
	network := settings.IP.Network()
	filter := makeIPFilter(network)

	return MakeJob(settings.jobSettings(), func() (common.MapStr, []TaskRunner, error) {
		// TODO: check for better DNS IP lookup support:
		//         - The net.LookupIP drops ipv6 zone index
		//
		resolveStart := time.Now()
		ips, err := net.LookupIP(host)
		if err != nil {
			return resolveErr(host, err)
		}

		resolveEnd := time.Now()
		resolveRTT := resolveEnd.Sub(resolveStart)

		if filter != nil {
			ips = filterIPs(ips, filter)
		}

		if len(ips) == 0 {
			err := fmt.Errorf("no %v address resolvable for host %v", network, host)
			return resolveErr(host, err)
		}

		// create ip ping tasks
		cont := make([]TaskRunner, len(ips))
		for i, ip := range ips {
			addr := &net.IPAddr{IP: ip}
			event := resolveIPEvent(host, ip.String(), resolveRTT)
			cont[i] = WithFields(event, pingFactory(addr))
		}
		return nil, cont, nil
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

func resolveErr(host string, err error) (common.MapStr, []TaskRunner, error) {
	event := common.MapStr{
		"monitor": common.MapStr{
			"host": host,
		},
		"resolve": common.MapStr{
			"host": host,
		},
	}
	return event, nil, err
}

// WithFields wraps a TaskRunner, updating all events returned with the set of
// fields configured.
func WithFields(fields common.MapStr, r TaskRunner) TaskRunner {
	return MakeCont(func() (common.MapStr, []TaskRunner, error) {
		event, cont, err := r.Run()
		if event != nil {
			event = event.Clone()
			event.DeepUpdate(fields)
		} else if err != nil {
			event = common.MapStr{}
			event.DeepUpdate(fields)
		}

		for i := range cont {
			cont[i] = WithFields(fields, cont[i])
		}
		return event, cont, err
	})
}

// WithDuration wraps a TaskRunner, measuring the duration between creation and
// finish of the actual task and sub-tasks.
func WithDuration(field string, r TaskRunner) TaskRunner {
	return MakeCont(func() (common.MapStr, []TaskRunner, error) {
		return withStart(field, time.Now(), r).Run()
	})
}

func withStart(field string, start time.Time, r TaskRunner) TaskRunner {
	return MakeCont(func() (common.MapStr, []TaskRunner, error) {
		event, cont, err := r.Run()
		if event != nil {
			event.Put(field, look.RTT(time.Since(start)))
		}

		for i := range cont {
			cont[i] = withStart(field, start, cont[i])
		}
		return event, cont, err
	})
}

func (f *funcJob) Name() string { return f.settings.Name }

func (f *funcJob) Run() (beat.Event, []JobRunner, error) { return f.run() }

func (f funcTask) Run() (common.MapStr, []TaskRunner, error) { return f.run() }

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

// MakeJobSetting creates a new JobSettings structure without any global event fields.
func MakeJobSetting(name string) JobSettings {
	return JobSettings{Name: name}
}

// WithFields adds new event fields to a Job. Existing fields will be
// overwritten.
// The fields map will be updated (no copy).
func (s JobSettings) WithFields(m common.MapStr) JobSettings {
	s.AddFields(m)
	return s
}

// AddFields adds new event fields to a Job. Existing fields will be
// overwritten.
func (s *JobSettings) AddFields(m common.MapStr) { addFields(&s.Fields, m) }

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

func (s *HostJobSettings) jobSettings() JobSettings {
	return JobSettings{Name: s.Name, Fields: s.Fields}
}
