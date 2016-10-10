package monitors

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/elastic/beats/libbeat/common"

	"github.com/elastic/beats/heartbeat/look"
)

type funcJob struct {
	name, typ string
	funcTask
}

type funcTask struct {
	run func() (common.MapStr, []TaskRunner, error)
}

type IPSettings struct {
	IPv4 bool     `config:"ipv4"`
	IPv6 bool     `config:"ipv6"`
	Mode PingMode `config:"mode"`
}

type PingMode uint8

const (
	PingModeUndefined PingMode = iota
	PingAny
	PingAll
)

var DefaultIPSettings = IPSettings{
	IPv4: true,
	IPv6: true,
	Mode: PingAny,
}

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

func MakeSimpleJob(name, typ string, f func() (common.MapStr, error)) Job {
	return MakeJob(name, typ, func() (common.MapStr, []TaskRunner, error) {
		event, err := f()
		return event, nil, err
	})
}

func MakeJob(name, typ string, f func() (common.MapStr, []TaskRunner, error)) Job {
	return &funcJob{name, typ, funcTask{func() (common.MapStr, []TaskRunner, error) {
		return annotated(time.Now(), typ, f).Run()
	}}}
}

func MakeCont(f func() (common.MapStr, []TaskRunner, error)) TaskRunner {
	return funcTask{f}
}

func MakeSimpleCont(f func() (common.MapStr, error)) TaskRunner {
	return MakeCont(func() (common.MapStr, []TaskRunner, error) {
		event, err := f()
		return event, nil, err
	})
}

func MakePingIPFactory(
	fields common.MapStr,
	f func(*net.IPAddr) (common.MapStr, error),
) func(*net.IPAddr) TaskRunner {
	return func(ip *net.IPAddr) TaskRunner {
		r := MakeSimpleCont(func() (common.MapStr, error) { return f(ip) })
		if len(fields) > 0 {
			r = WithFields(fields, r)
		}
		return r
	}
}

var emptyTask = MakeSimpleCont(func() (common.MapStr, error) { return nil, nil })

func MakePingAllIPFactory(
	fields common.MapStr,
	f func(*net.IPAddr) []func() (common.MapStr, error),
) func(*net.IPAddr) TaskRunner {
	makeTask := func(f func() (common.MapStr, error)) TaskRunner {
		if len(fields) > 0 {
			return WithFields(fields, MakeSimpleCont(f))
		}
		return MakeSimpleCont(f)
	}

	return func(ip *net.IPAddr) TaskRunner {
		cont := f(ip)
		switch len(cont) {
		case 0:
			return emptyTask
		case 1:
			return makeTask(cont[0])
		}

		tasks := make([]TaskRunner, len(cont))
		for i, c := range cont {
			tasks[i] = makeTask(c)
		}
		return MakeCont(func() (common.MapStr, []TaskRunner, error) {
			return nil, tasks, nil
		})
	}
}

func MakePingAllIPPortFactory(
	fields common.MapStr,
	ports []uint16,
	f func(*net.IPAddr, uint16) (common.MapStr, error),
) func(*net.IPAddr) TaskRunner {
	if len(ports) == 1 {
		port := ports[0]
		fields := fields.Clone()
		fields["port"] = strconv.Itoa(int(port))
		return MakePingIPFactory(fields, func(ip *net.IPAddr) (common.MapStr, error) {
			return f(ip, port)
		})
	}

	return MakePingAllIPFactory(fields, func(ip *net.IPAddr) []func() (common.MapStr, error) {
		funcs := make([]func() (common.MapStr, error), len(ports))
		for i := range ports {
			port := ports[i]
			funcs[i] = func() (common.MapStr, error) {
				event, err := f(ip, port)
				if event == nil {
					event = common.MapStr{}
				}
				event["port"] = strconv.Itoa(int(port))
				return event, err
			}
		}
		return funcs
	})
}

func MakeByIPJob(
	name, typ string,
	ip net.IP,
	pingFactory func(ip *net.IPAddr) TaskRunner,
) (Job, error) {
	// use ResolveIPAddr to parse the ip into net.IPAddr adding a zone info
	// if ipv6 is used.
	addr, err := net.ResolveIPAddr("ip", ip.String())
	if err != nil {
		return nil, err
	}

	fields := common.MapStr{"ip": addr.String()}
	return MakeJob(name, typ, WithFields(fields, pingFactory(addr)).Run), nil
}

func MakeByHostJob(
	name, typ string,
	host string,
	settings IPSettings,
	pingFactory func(ip *net.IPAddr) TaskRunner,
) (Job, error) {
	network := settings.Network()
	if network == "" {
		return nil, errors.New("pinging hosts requires ipv4 or ipv6 mode enabled")
	}

	mode := settings.Mode
	if mode == PingAny {
		return MakeJob(name, typ, func() (common.MapStr, []TaskRunner, error) {
			event := common.MapStr{"host": host}

			dnsStart := time.Now()
			ip, err := net.ResolveIPAddr(network, host)
			if err != nil {
				return event, nil, err
			}

			dnsEnd := time.Now()
			dnsRTT := dnsEnd.Sub(dnsStart)
			event["resolve_rtt"] = look.RTT(dnsRTT)
			event["ip"] = ip.String()

			return WithFields(event, pingFactory(ip)).Run()
		}), nil
	}

	filter := makeIPFilter(network)
	return MakeJob(name, typ, func() (common.MapStr, []TaskRunner, error) {
		event := common.MapStr{"host": host}

		// TODO: check for better DNS IP lookup support:
		//         - The net.LookupIP drops ipv6 zone index
		//
		dnsStart := time.Now()
		ips, err := net.LookupIP(host)
		if err != nil {
			return event, nil, err
		}

		dnsEnd := time.Now()
		dnsRTT := dnsEnd.Sub(dnsStart)

		event["resolve_rtt"] = look.RTT(dnsRTT)
		if filter != nil {
			ips = filterIPs(ips, filter)
		}

		if len(ips) == 0 {
			err := fmt.Errorf("no %v address resolvable for host %v", network, host)
			return event, nil, err
		}

		// create ip ping tasks
		cont := make([]TaskRunner, len(ips))
		for i, ip := range ips {
			addr := &net.IPAddr{IP: ip}
			fields := event.Clone()
			fields["ip"] = ip.String()
			cont[i] = WithFields(fields, pingFactory(addr))
		}
		return nil, cont, nil
	}), nil
}

func WithFields(fields common.MapStr, r TaskRunner) TaskRunner {
	return MakeCont(func() (common.MapStr, []TaskRunner, error) {
		event, cont, err := r.Run()
		if event == nil {
			event = common.MapStr{}
		}
		event.Update(fields)

		for i := range cont {
			cont[i] = WithFields(fields, cont[i])
		}
		return event, cont, err
	})
}

func WithDuration(name string, r TaskRunner) TaskRunner {
	return MakeCont(func() (common.MapStr, []TaskRunner, error) {
		return withStart(name, time.Now(), r).Run()
	})
}

func withStart(field string, start time.Time, r TaskRunner) TaskRunner {
	return MakeCont(func() (common.MapStr, []TaskRunner, error) {
		event, cont, err := r.Run()
		if event != nil {
			event[field] = look.RTT(time.Now().Sub(start))
		}

		for i := range cont {
			cont[i] = withStart(field, start, cont[i])
		}
		return event, cont, err
	})
}

func (f *funcJob) Name() string { return f.name }

func (f funcTask) Run() (common.MapStr, []TaskRunner, error) { return f.run() }

func (f funcTask) annotated(start time.Time, typ string) TaskRunner {
	return annotated(start, typ, f.run)
}

func (p *PingMode) Unpack(v interface{}) error {
	var fail = errors.New("expecting 'any' or 'all'")
	s, ok := v.(string)
	if !ok {
		return fail
	}

	switch s {
	case "all":
		*p = PingAll
	case "any":
		*p = PingAny
	default:
		return fail
	}
	return nil
}

func annotated(start time.Time, typ string, fn func() (common.MapStr, []TaskRunner, error)) TaskRunner {
	return MakeCont(func() (common.MapStr, []TaskRunner, error) {
		event, cont, err := fn()
		if err != nil {
			if event == nil {
				event = common.MapStr{}
			}
			event["error"] = look.Reason(err)
		}

		if event != nil {
			event.Update(common.MapStr{
				"@timestamp": look.Timestamp(start),
				"duration":   look.RTT(time.Now().Sub(start)),
				"type":       typ,
				"up":         err == nil,
			})
		}

		for i := range cont {
			if fcont, ok := cont[i].(funcTask); ok {
				cont[i] = fcont.annotated(start, typ)
			} else {
				cont[i] = annotated(start, typ, cont[i].Run)
			}
		}
		return event, cont, nil
	})
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
