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
	"regexp"
	"sync"
	"time"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/metric/system/host"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/go-sysinfo"
)

func init() {
	processors.RegisterPlugin("add_host_metadata", New)
}

type addHostMetadata struct {
	lastUpdate struct {
		time.Time
		sync.Mutex
	}
	data    common.MapStrPointer
	geoData common.MapStr
	config  Config
}

const (
	processorName = "add_host_metadata"
)

// New constructs a new add_host_metadata processor.
func New(cfg *common.Config) (processors.Processor, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, errors.Wrapf(err, "fail to unpack the %v configuration", processorName)
	}

	p := &addHostMetadata{
		config: config,
		data:   common.NewMapStrPointer(nil),
	}
	p.loadData()

	if config.Geo != nil {
		if len(config.Geo.Location) > 0 {
			// Regexp matching a number with an optional decimal component
			// Valid numbers: '123', '123.23', etc.
			latOrLon := `\-?\d+(\.\d+)?`

			// Regexp matching a pair of lat lon coordinates.
			// e.g. 40.123, -92.929
			locRegexp := `^\s*` + // anchor to start of string with optional whitespace
				latOrLon + // match the latitude
				`\s*\,\s*` + // match the separator. optional surrounding whitespace
				latOrLon + // match the longitude
				`\s*$` //optional whitespace then end anchor

			if m, _ := regexp.MatchString(locRegexp, config.Geo.Location); !m {
				return nil, errors.New(fmt.Sprintf("Invalid lat,lon  string for add_host_metadata: %s", config.Geo.Location))
			}
		}

		geoFields := common.MapStr{
			"name":             config.Geo.Name,
			"location":         config.Geo.Location,
			"continent_name":   config.Geo.ContinentName,
			"country_iso_code": config.Geo.CountryISOCode,
			"region_name":      config.Geo.RegionName,
			"region_iso_code":  config.Geo.RegionISOCode,
			"city_name":        config.Geo.CityName,
		}
		// Delete any empty values
		blankStringMatch := regexp.MustCompile(`^\s*$`)
		for k, v := range geoFields {
			vStr := v.(string)
			if blankStringMatch.MatchString(vStr) {
				delete(geoFields, k)
			}
		}
		p.geoData = common.MapStr{"host": common.MapStr{"geo": geoFields}}
	}

	return p, nil
}

// Run enriches the given event with the host meta data
func (p *addHostMetadata) Run(event *beat.Event) (*beat.Event, error) {
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

func (p *addHostMetadata) expired() bool {
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

func (p *addHostMetadata) loadData() error {
	if !p.expired() {
		return nil
	}

	h, err := sysinfo.Host()
	if err != nil {
		return err
	}

	data := host.MapHostInfo(h.Info())
	if p.config.NetInfoEnabled {
		// IP-address and MAC-address
		var ipList, hwList, err = p.getNetInfo()
		if err != nil {
			logp.Info("Error when getting network information %v", err)
		}

		if len(ipList) > 0 {
			data.Put("host.ip", ipList)
		}
		if len(hwList) > 0 {
			data.Put("host.mac", hwList)
		}
	}

	if p.config.Name != "" {
		data.Put("host.name", p.config.Name)
	}
	p.data.Set(data)
	return nil
}

func (p *addHostMetadata) getNetInfo() ([]string, []string, error) {
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

func (p *addHostMetadata) String() string {
	return fmt.Sprintf("%v=[netinfo.enabled=[%v], cache.ttl=[%v]]",
		processorName, p.config.NetInfoEnabled, p.config.CacheTTL)
}
