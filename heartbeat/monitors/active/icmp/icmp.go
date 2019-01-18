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
	"net/url"

	"github.com/elastic/beats/heartbeat/eventext"
	"github.com/elastic/beats/heartbeat/look"
	"github.com/elastic/beats/heartbeat/monitors"
	"github.com/elastic/beats/heartbeat/monitors/jobs"
	"github.com/elastic/beats/heartbeat/monitors/wrappers"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

func init() {
	monitors.RegisterActive("icmp", create)
}

var debugf = logp.MakeDebug("icmp")

func create(
	name string,
	cfg *common.Config,
) (jobs []jobs.Job, endpoints int, err error) {
	config := DefaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, 0, err
	}

	// TODO: check icmp is support by OS + check we've
	// got required credentials (implementation uses RAW socket, requires root +
	// not supported on all OSes)
	// TODO: replace icmp package base reader/sender using raw sockets with
	//       OS specific solution

	ipVersion := config.Mode.Network()
	if len(config.Hosts) > 0 && ipVersion == "" {
		err := fmt.Errorf("pinging hosts requires ipv4 or ipv6 mode enabled")
		return nil, 0, err
	}

	var loopErr error
	loopInit.Do(func() {
		debugf("initialize icmp handler")
		loop, loopErr = newICMPLoop()
	})
	if loopErr != nil {
		debugf("Failed to initialize ICMP loop %v", loopErr)
		return nil, 0, loopErr
	}

	if err := loop.checkNetworkMode(ipVersion); err != nil {
		return nil, 0, err
	}

	pingFactory := monitors.MakePingIPFactory(createPingIPFactory(&config))

	for _, host := range config.Hosts {
		settings := monitors.MakeHostJobSettings(host, config.Mode)
		job, err := monitors.MakeByHostJob(settings, pingFactory)

		if err != nil {
			return nil, 0, err
		}

		u, err := url.Parse(fmt.Sprintf("icmp://%s", host))
		if err != nil {
			return nil, 0, err
		}

		jobs = append(jobs, wrappers.WithURLField(u, job))
	}

	return jobs, len(config.Hosts), nil
}

func createPingIPFactory(config *Config) func(*beat.Event, *net.IPAddr) error {
	return func(event *beat.Event, ip *net.IPAddr) error {
		rtt, n, err := loop.ping(ip, config.Timeout, config.Wait)
		if err != nil {
			return err
		}

		icmpFields := common.MapStr{"requests": n}
		if err == nil {
			icmpFields["rtt"] = look.RTT(rtt)
			eventext.MergeEventFields(event, icmpFields)
		}

		return nil
	}
}
