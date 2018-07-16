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

package add_host_metadata

import (
	"fmt"
	"net"
	"time"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/metric/system/host"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/go-sysinfo"
	"github.com/elastic/go-sysinfo/types"
)

func init() {
	processors.RegisterPlugin("add_host_metadata", newHostMetadataProcessor)
}

type addHostMetadata struct {
	info       types.HostInfo
	lastUpdate time.Time
	data       common.MapStr
	config     Config
}

const (
	processorName   = "add_host_metadata"
	cacheExpiration = time.Minute * 5
)

func newHostMetadataProcessor(cfg *common.Config) (processors.Processor, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, errors.Wrapf(err, "fail to unpack the %v configuration", processorName)
	}

	h, err := sysinfo.Host()
	if err != nil {
		return nil, err
	}
	p := &addHostMetadata{
		info:   h.Info(),
		config: config,
	}
	return p, nil
}

// Run enriches the given event with the host meta data
func (p *addHostMetadata) Run(event *beat.Event) (*beat.Event, error) {
	p.loadData()
	event.Fields.DeepUpdate(p.data)
	return event, nil
}

func (p *addHostMetadata) loadData() {

	// Check if cache is expired
	if p.lastUpdate.Add(cacheExpiration).Before(time.Now()) {
		p.data = host.MapHostInfo(p.info)

		if p.config.NetInfoEnabled {
			// IP-address and MAC-address
			var ipList, hwList, err = p.getNetInfo()
			if err != nil {
				logp.Info("Error when getting network information %v", err)
			}

			if len(ipList) > 0 {
				p.data.Put("host.ip", ipList)
			}
			if len(hwList) > 0 {
				p.data.Put("host.mac", hwList)
			}
		}
		p.lastUpdate = time.Now()
	}
}

func (p addHostMetadata) getNetInfo() ([]string, []string, error) {
	var ipList []string
	var hwList []string

	// Get all interfaces and loop through them
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, nil, err
	}

	// Keep track of all errors
	var errs multierror.Errors

	for _, i := range ifaces {
		// Skip loopback interfaces
		if i.Flags&net.FlagLoopback == net.FlagLoopback {
			continue
		}

		hw := i.HardwareAddr.String()
		// Skip empty hardware addresses
		if hw != "" {
			hwList = append(hwList, hw)
		}

		addrs, err := i.Addrs()
		if err != nil {
			// If we get an error, keep track of it and continue with the next interface
			errs = append(errs, err)
			continue
		}

		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				ipList = append(ipList, v.IP.String())
			case *net.IPAddr:
				ipList = append(ipList, v.IP.String())
			}
		}
	}

	return ipList, hwList, errs.Err()
}

func (p addHostMetadata) String() string {
	return fmt.Sprintf("%v=[netinfo.enabled=[%v]]",
		processorName, p.config.NetInfoEnabled)
}
