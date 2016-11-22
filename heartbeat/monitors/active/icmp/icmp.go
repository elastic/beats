package icmp

import (
	"fmt"
	"net"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/heartbeat/look"
	"github.com/elastic/beats/heartbeat/monitors"
)

func init() {
	monitors.RegisterActive("icmp", create)
}

var debugf = logp.MakeDebug("icmp")

func create(
	info monitors.Info,
	cfg *common.Config,
) ([]monitors.Job, error) {
	config := DefaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	// TODO: check icmp is support by OS + check we've
	// got required credentials (implementation uses RAW socket, requires root +
	// not supported on all OSes)
	// TODO: replace icmp package base reader/sender using raw sockets with
	//       OS specific solution

	var jobs []monitors.Job
	addJob := func(t monitors.Job, err error) error {
		if err != nil {
			return err
		}
		jobs = append(jobs, t)
		return nil
	}

	ipVersion := config.Mode.Network()
	if len(config.Hosts) > 0 && ipVersion == "" {
		err := fmt.Errorf("pinging hosts requires ipv4 or ipv6 mode enabled")
		return nil, err
	}

	var loopErr error
	loopInit.Do(func() {
		debugf("initialize icmp handler")
		loop, loopErr = newICMPLoop()
	})
	if loopErr != nil {
		debugf("Failed to initialize ICMP loop %v", loopErr)
		return nil, loopErr
	}

	typ := config.Name
	network := config.Mode.Network()
	pingFactory := monitors.MakePingIPFactory(nil, createPingIPFactory(&config))

	for _, host := range config.Hosts {
		ip := net.ParseIP(host)
		if ip != nil {
			name := fmt.Sprintf("icmp-ip@%v", ip.String())
			err := addJob(monitors.MakeByIPJob(name, typ, ip, pingFactory))
			if err != nil {
				return nil, err
			}
			continue
		}

		name := fmt.Sprintf("%v-host-%v@%v", config.Name, network, host)
		err := addJob(monitors.MakeByHostJob(name, typ, host, config.Mode, pingFactory))
		if err != nil {
			return nil, err
		}
	}

	return jobs, nil
}

func createPingIPFactory(config *Config) func(*net.IPAddr) (common.MapStr, error) {
	return func(ip *net.IPAddr) (common.MapStr, error) {
		rtt, _, err := loop.ping(ip, config.Timeout, config.Wait)
		if err != nil {
			return nil, err
		}

		return common.MapStr{
			"icmp_rtt": look.RTT(rtt),
		}, nil
	}
}
