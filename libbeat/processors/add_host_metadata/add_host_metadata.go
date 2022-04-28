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
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/metric/system/host"
	"github.com/elastic/beats/v7/libbeat/processors"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor"
	"github.com/elastic/beats/v7/libbeat/processors/util"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-sysinfo"
)

func init() {
	processors.RegisterPlugin("add_host_metadata", New)
	jsprocessor.RegisterPlugin("AddHostMetadata", New)
}

type addHostMetadata struct {
	lastUpdate struct {
		time.Time
		sync.Mutex
	}
	data    mapstr.Pointer
	geoData mapstr.M
	config  Config
	logger  *logp.Logger
}

const (
	processorName = "add_host_metadata"
)

// New constructs a new add_host_metadata processor.
func New(cfg *config.C) (processors.Processor, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, errors.Wrapf(err, "fail to unpack the %v configuration", processorName)
	}

	p := &addHostMetadata{
		config: config,
		data:   mapstr.NewPointer(nil),
		logger: logp.NewLogger("add_host_metadata"),
	}
	p.loadData()

	if config.Geo != nil {
		geoFields, err := util.GeoConfigToMap(*config.Geo)
		if err != nil {
			return nil, err
		}
		p.geoData = mapstr.M{"host": mapstr.M{"geo": geoFields}}
	}

	return p, nil
}

// Run enriches the given event with the host meta data
func (p *addHostMetadata) Run(event *beat.Event) (*beat.Event, error) {
	// check replace_host_fields field
	if !p.config.ReplaceFields && skipAddingHostMetadata(event) {
		return event, nil
	}

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
		var ipList, hwList, err = util.GetNetInfo()
		if err != nil {
			p.logger.Infof("Error when getting network information %v", err)
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

func (p *addHostMetadata) String() string {
	return fmt.Sprintf("%v=[netinfo.enabled=[%v], cache.ttl=[%v]]",
		processorName, p.config.NetInfoEnabled, p.config.CacheTTL)
}

func skipAddingHostMetadata(event *beat.Event) bool {
	// If host fields exist(besides host.name added by libbeat) in event, skip add_host_metadata.
	hostFields, err := event.Fields.GetValue("host")

	// Don't skip if there are no fields
	if err != nil || hostFields == nil {
		return false
	}

	switch m := hostFields.(type) {
	case mapstr.M:
		// if "name" is the only field, don't skip
		hasName, _ := m.HasKey("name")
		if hasName && len(m) == 1 {
			return false
		}
		return true
	case map[string]interface{}:
		hostMapStr := mapstr.M(m)
		// if "name" is the only field, don't skip
		hasName, _ := hostMapStr.HasKey("name")
		if hasName && len(m) == 1 {
			return false
		}
		return true
	case map[string]string:
		// if "name" is the only field, don't skip
		if m["name"] != "" && len(m) == 1 {
			return false
		}
		return true
	default:
		return false
	}
}
