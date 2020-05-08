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

	"github.com/elastic/beats/v7/heartbeat/eventext"
	"github.com/elastic/beats/v7/heartbeat/look"
	"github.com/elastic/beats/v7/heartbeat/monitors"
	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

var debugf = logp.MakeDebug("icmp")

func init() {
	monitors.RegisterActive("icmp", create)
}

func create(
	name string,
	commonConfig *common.Config,
) (jobs []jobs.Job, endpoints int, err error) {
	loop, err := getStdLoop()
	if err != nil {
		logp.Warn("Failed to initialize ICMP loop %v", err)
		return nil, 0, err
	}

	config := DefaultConfig
	if err := commonConfig.Unpack(&config); err != nil {
		return nil, 0, err
	}

	jf, err := newJobFactory(config, monitors.NewStdResolver(), loop)
	if err != nil {
		return nil, 0, err
	}
	return jf.makeJobs()

}

type jobFactory struct {
	config    Config
	resolver  monitors.Resolver
	loop      ICMPLoop
	ipVersion string
}

func newJobFactory(config Config, resolver monitors.Resolver, loop ICMPLoop) (*jobFactory, error) {
	jf := &jobFactory{config: config, resolver: resolver, loop: loop}
	err := jf.checkConfig()
	if err != nil {
		return nil, err
	}

	return jf, nil
}

func (jf *jobFactory) checkConfig() error {
	jf.ipVersion = jf.config.Mode.Network()
	if len(jf.config.Hosts) > 0 && jf.ipVersion == "" {
		err := fmt.Errorf("pinging hosts requires ipv4 or ipv6 mode enabled")
		return err
	}

	return nil
}

func (jf *jobFactory) makeJobs() (j []jobs.Job, endpoints int, err error) {
	if err := jf.loop.checkNetworkMode(jf.ipVersion); err != nil {
		return nil, 0, err
	}

	pingFactory := jf.pingIPFactory(&jf.config)

	for _, host := range jf.config.Hosts {
		job, err := monitors.MakeByHostJob(host, jf.config.Mode, monitors.NewStdResolver(), pingFactory)

		if err != nil {
			return nil, 0, err
		}

		u, err := url.Parse(fmt.Sprintf("icmp://%s", host))
		if err != nil {
			return nil, 0, err
		}

		j = append(j, wrappers.WithURLField(u, job))
	}

	return j, len(jf.config.Hosts), nil
}

func (jf *jobFactory) pingIPFactory(config *Config) func(*net.IPAddr) jobs.Job {
	return monitors.MakePingIPFactory(func(event *beat.Event, ip *net.IPAddr) error {
		rtt, n, err := jf.loop.ping(ip, config.Timeout, config.Wait)
		if err != nil {
			return err
		}

		icmpFields := common.MapStr{"requests": n}
		if err == nil {
			icmpFields["rtt"] = look.RTT(rtt)
			eventext.MergeEventFields(event, common.MapStr{"icmp": icmpFields})
		}

		return nil
	})
}
