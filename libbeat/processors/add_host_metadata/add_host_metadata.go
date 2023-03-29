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

	"github.com/elastic/elastic-agent-libs/monitoring"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/features"
	"github.com/elastic/beats/v7/libbeat/processors"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor"
	"github.com/elastic/beats/v7/libbeat/processors/util"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/host"
	"github.com/elastic/go-sysinfo"
)

const processorName = "add_host_metadata"
const logName = "processor." + processorName

var (
	reg *monitoring.Registry
)

func init() {
	processors.RegisterPlugin(processorName, New)
	jsprocessor.RegisterPlugin("AddHostMetadata", New)

	reg = monitoring.Default.NewRegistry(logName, monitoring.DoNotReport)
}

type metrics struct {
	FQDNLookupFailed *monitoring.Int
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
	metrics metrics
}

// New constructs a new add_host_metadata processor.
func New(cfg *config.C) (processors.Processor, error) {
	c := defaultConfig()
	if err := cfg.Unpack(&c); err != nil {
		return nil, fmt.Errorf("fail to unpack the %v configuration: %w", processorName, err)
	}

	p := &addHostMetadata{
		config: c,
		data:   mapstr.NewPointer(nil),
		logger: logp.NewLogger(logName),
		metrics: metrics{
			FQDNLookupFailed: monitoring.NewInt(reg, "fqdn_lookup_failed"),
		},
	}
	if err := p.loadData(); err != nil {
		return nil, fmt.Errorf("failed to load data: %w", err)
	}

	if c.Geo != nil {
		geoFields, err := util.GeoConfigToMap(*c.Geo)
		if err != nil {
			return nil, err
		}
		p.geoData = mapstr.M{"host": mapstr.M{"geo": geoFields}}
	}

	err := features.AddFQDNOnChangeCallback(p.handleFQDNReportingChange, processorName)
	if err != nil {
		return nil, fmt.Errorf(
			"could not register callback for FQDN reporting onChange from %s processor: %w",
			processorName, err,
		)
	}

	return p, nil
}

// Run enriches the given event with the host metadata
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

// Ideally we'd be able to implement the Closer interface here and
// deregister the callback.  But processors that can be used with the
// `script` processor are not allowed to implement the Closer
// interface (@see https://github.com/elastic/beats/pull/16349).
//func (p *addHostMetadata) Close() error {
//	features.RemoveFQDNOnChangeCallback(processorName)
//	return nil
//}

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

	hostname := h.Info().Hostname
	if features.FQDN() {
		fqdn, err := h.FQDN()
		if err != nil {
			// FQDN lookup is "best effort". If it fails, we monitor the failure, fallback to
			// the OS-reported hostname, and move on.
			p.metrics.FQDNLookupFailed.Inc()
			p.logger.Debugf(
				"unable to lookup FQDN (failed attempt counter: %d): %s, using hostname = %s as FQDN",
				p.metrics.FQDNLookupFailed.Get(),
				err.Error(),
				hostname,
			)
		} else {
			hostname = fqdn
		}
	}

	data := host.MapHostInfo(h.Info(), hostname)
	if p.config.NetInfoEnabled {
		// IP-address and MAC-address
		var ipList, hwList, err = util.GetNetInfo()
		if err != nil {
			p.logger.Infof("Error when getting network information %v", err)
		}

		if len(ipList) > 0 {
			if _, err := data.Put("host.ip", ipList); err != nil {
				return fmt.Errorf("could not set host.ip: %w", err)
			}
		}
		if len(hwList) > 0 {
			if _, err := data.Put("host.mac", hwList); err != nil {
				return fmt.Errorf("could not set host.mac: %w", err)
			}
		}
	}

	if p.config.Name != "" {
		if _, err := data.Put("host.name", p.config.Name); err != nil {
			return fmt.Errorf("could not set host.name: %w", err)
		}
	}

	p.data.Set(data)
	return nil
}

func (p *addHostMetadata) String() string {
	return fmt.Sprintf("%v=[netinfo.enabled=[%v], cache.ttl=[%v]]",
		processorName, p.config.NetInfoEnabled, p.config.CacheTTL)
}

func (p *addHostMetadata) handleFQDNReportingChange(new, old bool) {
	if new == old {
		// Nothing to do
		return
	}

	// Whether we should report the FQDN or not has changed.  Expire cache
	// so we start report the desired hostname value immediately.
	p.expireCache()
}

func (p *addHostMetadata) expireCache() {
	if p.config.CacheTTL <= 0 {
		return
	}

	p.lastUpdate.Lock()
	defer p.lastUpdate.Unlock()

	// Update cache's last updated timestamp to be zero,
	// effectively expiring the cache immediately.
	p.lastUpdate.Time = time.Time{}
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
