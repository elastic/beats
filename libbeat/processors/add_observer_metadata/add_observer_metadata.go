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

package add_observer_metadata

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/libbeat/processors/util"
	"github.com/elastic/go-sysinfo"
)

func init() {
	processors.RegisterPlugin("add_observer_metadata", New)
}

type observerMetadata struct {
	lastUpdate struct {
		time.Time
		sync.Mutex
	}
	data    common.MapStrPointer
	geoData common.MapStr
	config  Config
}

const (
	processorName = "add_observer_metadata"
)

func New(cfg *common.Config) (processors.Processor, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, errors.Wrapf(err, "fail to unpack the %v configuration", processorName)
	}

	p := &observerMetadata{
		config: config,
		data:   common.NewMapStrPointer(nil),
	}
	p.loadData()

	if config.Geo != nil {
		geoFields, err := util.GeoConfigToMap(*config.Geo)
		if err != nil {
			return nil, err
		}

		p.geoData = common.MapStr{"observer": common.MapStr{"geo": geoFields}}
	}

	return p, nil
}

// Run enriches the given event with the observer meta data
func (p *observerMetadata) Run(event *beat.Event) (*beat.Event, error) {
	err := p.loadData()
	if err != nil {
		return nil, err
	}

	event.Fields.DeepUpdate(p.data.Get().Clone())

	if len(p.geoData) > 0 {
		event.Fields.DeepUpdate(p.geoData)
	}
	return event, nil
}

func (p *observerMetadata) expired() bool {
	if p.config.CacheTTL <= 0 {
		return true
	}

	p.lastUpdate.Lock()
	defer p.lastUpdate.Unlock()

	if p.lastUpdate.Add(p.config.CacheTTL).After(time.Now()) {
		return false
	}
	p.lastUpdate.Time = time.Now()
	return true
}

func (p *observerMetadata) loadData() error {
	if !p.expired() {
		return nil
	}

	h, err := sysinfo.Host()
	if err != nil {
		return err
	}

	hostInfo := h.Info()
	data := common.MapStr{
		"observer": common.MapStr{
			"hostname": hostInfo.Hostname,
			"type":     "heartbeat",
			"vendor":   "elastic",
		},
	}
	if p.config.NetInfoEnabled {
		// IP-address and MAC-address
		var ipList, hwList, err = p.getNetInfo()
		if err != nil {
			logp.Info("Error when getting network information %v", err)
		}

		if len(ipList) > 0 {
			data.Put("observer.ip", ipList)
		}
		if len(hwList) > 0 {
			data.Put("observer.mac", hwList)
		}
	}

	p.data.Set(data)
	return nil
}

func (p *observerMetadata) getNetInfo() ([]string, []string, error) {
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

func (p *observerMetadata) String() string {
	return fmt.Sprintf("%v=[netinfo.enabled=[%v], cache.ttl=[%v]]",
		processorName, p.config.NetInfoEnabled, p.config.CacheTTL)
}
