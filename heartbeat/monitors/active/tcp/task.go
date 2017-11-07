package tcp

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs/transport"

	"github.com/elastic/beats/heartbeat/look"
	"github.com/elastic/beats/heartbeat/monitors"
	"github.com/elastic/beats/heartbeat/monitors/active/dialchain"
	"github.com/elastic/beats/heartbeat/reason"
)

func newTCPMonitorHostJob(
	scheme, host string, port uint16,
	tls *transport.TLSConfig,
	config *Config,
) (monitors.Job, error) {
	typ := config.Name
	timeout := config.Timeout
	jobName := jobName(typ, jobType(scheme), host, []uint16{port})
	validator := makeValidateConn(config)
	pingAddr := net.JoinHostPort(host, strconv.Itoa(int(port)))

	taskDialer, err := buildDialerChain(scheme, tls, config)
	if err != nil {
		return nil, err
	}

	settings := monitors.MakeJobSetting(jobName).WithFields(common.MapStr{
		"monitor": common.MapStr{
			"host":   host,
			"scheme": scheme,
		},
		"tcp": common.MapStr{
			"port": port,
		},
	})

	return monitors.MakeSimpleJob(settings, func() (common.MapStr, error) {
		event := common.MapStr{}
		dialer, err := taskDialer.Build(event)
		if err != nil {
			return event, err
		}

		results, err := pingHost(dialer, pingAddr, timeout, validator)
		event.DeepUpdate(results)
		return event, err
	}), nil
}

func newTCPMonitorIPsJob(
	addr connURL,
	tls *transport.TLSConfig,
	config *Config,
) (monitors.Job, error) {
	typ := config.Name
	timeout := config.Timeout
	jobType := jobType(addr.Scheme)
	jobName := jobName(typ, jobType, addr.Host, addr.Ports)
	validator := makeValidateConn(config)

	dialerFactory, err := buildHostDialerChainFactory(addr.Scheme, tls, config)
	if err != nil {
		return nil, err
	}

	settings := monitors.MakeHostJobSettings(jobName, addr.Host, config.Mode)
	settings = settings.WithFields(common.MapStr{
		"monitor": common.MapStr{
			"scheme": addr.Scheme,
		},
	})

	debugf("Make TCP job: %v:%v", addr.Host, addr.Ports)
	pingFactory := createPingFactory(dialerFactory, addr, timeout, validator)
	return monitors.MakeByHostJob(settings, pingFactory)
}

func createPingFactory(
	makeDialerChain func(string) *dialchain.DialerChain,
	addr connURL,
	timeout time.Duration,
	validator ConnCheck,
) func(*net.IPAddr) monitors.TaskRunner {
	return monitors.MakePingAllIPPortFactory(addr.Ports,
		func(ip *net.IPAddr, port uint16) (common.MapStr, error) {
			ipStr := ip.String()
			host := net.JoinHostPort(ipStr, strconv.Itoa(int(port)))
			pingAddr := net.JoinHostPort(addr.Host, strconv.Itoa(int(port)))

			event := common.MapStr{
				"tcp": common.MapStr{
					"port": port,
				},
			}

			dialer, err := makeDialerChain(host).Build(event)
			if err != nil {
				return event, err
			}

			results, err := pingHost(dialer, pingAddr, timeout, validator)
			event.DeepUpdate(results)
			return event, err
		})
}

func pingHost(
	dialer transport.Dialer,
	host string,
	timeout time.Duration,
	validator ConnCheck,
) (common.MapStr, reason.Reason) {
	start := time.Now()
	deadline := start.Add(timeout)

	conn, err := dialer.Dial("tcp", host)
	if err != nil {
		debugf("dial failed with: %v", err)
		return nil, reason.IOFailed(err)
	}
	defer conn.Close()
	if validator == nil {
		// no additional validation step => ping success
		return common.MapStr{}, nil
	}

	if err := conn.SetDeadline(deadline); err != nil {
		debugf("setting connection deadline failed with: %v", err)
		return nil, reason.IOFailed(err)
	}

	validateStart := time.Now()
	err = validator.Validate(conn)
	if err != nil && err != errRecvMismatch {
		debugf("check failed with: %v", err)
		return nil, reason.IOFailed(err)
	}

	end := time.Now()
	event := common.MapStr{
		"tcp": common.MapStr{
			"rtt": common.MapStr{
				"validate": look.RTT(end.Sub(validateStart)),
			},
		},
	}
	if err != nil {
		event["error"] = reason.FailValidate(err)
	}
	return event, nil
}

func isTLSAddr(scheme string) bool {
	return scheme == "tls" || scheme == "ssl"
}

func jobType(scheme string) string {
	switch scheme {
	case "tls", "ssl":
		return scheme
	}
	return "plain"
}

func jobName(typ, jobType, host string, ports []uint16) string {
	var h string
	if len(ports) == 1 {
		h = fmt.Sprintf("%v:%v", host, ports[0])
	} else {
		h = fmt.Sprintf("%v:%v", host, ports)
	}
	return fmt.Sprintf("%v-%v@%v", typ, jobType, h)
}

func buildDialerChain(
	scheme string,
	tls *transport.TLSConfig,
	config *Config,
) (*dialchain.DialerChain, error) {
	d := &dialchain.DialerChain{
		Net: dialchain.TCPDialer(config.Timeout),
	}

	withProxy := config.Socks5.URL != ""
	if withProxy {
		d.AddLayer(dialchain.SOCKS5Layer(&config.Socks5))
	}

	// insert empty placeholder, so address can be replaced in dialer chain
	// by replacing this placeholder dialer
	d.AddLayer(dialchain.IDLayer())

	if isTLSAddr(scheme) {
		d.AddLayer(dialchain.TLSLayer(tls, config.Timeout))
	}

	if err := d.TestBuild(); err != nil {
		return nil, err
	}
	return d, nil
}

func buildHostDialerChainFactory(
	scheme string,
	tls *transport.TLSConfig,
	config *Config,
) (func(string) *dialchain.DialerChain, error) {
	template, err := buildDialerChain(scheme, tls, config)
	if err != nil {
		return nil, err
	}

	withProxy := config.Socks5.URL != ""
	addrIndex := 0
	if withProxy {
		addrIndex = 1
	}

	return func(addr string) *dialchain.DialerChain {
		// replace IDLayer placeholder in template with ConstAddrLayer
		d := template.Clone()
		d.Layers[addrIndex] = dialchain.ConstAddrLayer(addr)
		return d
	}, nil
}
