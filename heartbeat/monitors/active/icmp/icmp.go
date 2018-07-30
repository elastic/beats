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

	if err := loop.checkNetworkMode(ipVersion); err != nil {
		return nil, err
	}

	network := config.Mode.Network()
	pingFactory := monitors.MakePingIPFactory(createPingIPFactory(&config))

	for _, host := range config.Hosts {
		jobName := fmt.Sprintf("icmp-%v-host-%v@%v", config.Name, network, host)
		if ip := net.ParseIP(host); ip != nil {
			jobName = fmt.Sprintf("icmp-%v-ip@%v", config.Name, ip.String())
		}

		settings := monitors.MakeHostJobSettings(jobName, host, config.Mode)
		err := addJob(monitors.MakeByHostJob(settings, pingFactory))
		if err != nil {
			return nil, err
		}
	}

	return jobs, nil
}

func createPingIPFactory(config *Config) func(*net.IPAddr) (common.MapStr, error) {
	return func(ip *net.IPAddr) (common.MapStr, error) {
		rtt, n, err := loop.ping(ip, config.Timeout, config.Wait)

		fields := common.MapStr{"requests": n}
		if err == nil {
			fields["rtt"] = look.RTT(rtt)
		}

		event := common.MapStr{"icmp": fields}
		return event, err
	}
}
