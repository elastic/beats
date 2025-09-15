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
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/go-sysinfo"
	"github.com/elastic/go-sysinfo/types"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/features"
	"github.com/elastic/beats/v7/libbeat/processors"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor/registry"
	"github.com/elastic/beats/v7/libbeat/processors/util"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/host"
)

const processorName = "add_host_metadata"
const logName = "processor." + processorName

var (
	reg *monitoring.Registry
)

func init() {
	processors.RegisterPlugin(processorName, New)
	jsprocessor.RegisterPlugin("AddHostMetadata", New)

	reg = monitoring.Default.GetOrCreateRegistry(logName, monitoring.DoNotReport)
}

type metrics struct {
	FQDNLookupFailed *monitoring.Int
}

// Interfaces to make mocking getting the hostname easier
type hostInfo interface {
	Info() types.HostInfo
	FQDNWithContext(context.Context) (string, error)
}

type hostInfoFactory func() (hostInfo, error)

type hostMetadataCache struct {
	sync.Mutex
	lastUpdate time.Time
	data       mapstr.Pointer
}

type addHostMetadata struct {
	// One cache for standard hostname, one for FQDN
	caches          [2]hostMetadataCache
	geoData         mapstr.M
	config          Config
	logger          *logp.Logger
	metrics         metrics
	hostInfoFactory hostInfoFactory
}

// New constructs a new add_host_metadata processor.
func New(cfg *config.C, log *logp.Logger) (beat.Processor, error) {
	c := defaultConfig()
	if err := cfg.Unpack(&c); err != nil {
		return nil, fmt.Errorf("fail to unpack the %v configuration: %w", processorName, err)
	}

	p := &addHostMetadata{
		caches: [2]hostMetadataCache{
			{data: mapstr.NewPointer(nil)},
			{data: mapstr.NewPointer(nil)},
		},
		config: c,
		logger: log.Named(logName),
		metrics: metrics{
			FQDNLookupFailed: monitoring.NewInt(reg, "fqdn_lookup_failed"),
		},
		hostInfoFactory: func() (hostInfo, error) { return sysinfo.Host() },
	}
	// Fetch and cache the initial host data.
	if _, err := p.loadData(features.FQDN()); err != nil {
		return nil, fmt.Errorf("failed to load data: %w", err)
	}

	if c.Geo != nil {
		geoFields, err := util.GeoConfigToMap(*c.Geo)
		if err != nil {
			return nil, err
		}
		p.geoData = mapstr.M{"host": mapstr.M{"geo": geoFields}}
	}

	return p, nil
}

// Run enriches the given event with the host metadata
func (p *addHostMetadata) Run(event *beat.Event) (*beat.Event, error) {
	// check replace_host_fields field
	if !p.config.ReplaceFields && skipAddingHostMetadata(event) {
		return event, nil
	}

	data, err := p.loadData(features.FQDN())
	if err != nil {
		return nil, fmt.Errorf("error loading data during event update: %w", err)
	}

	// Superficially this clone seems unnecessary, but it seems to have been
	// applied as a fix a long time ago -- possibly there can be later processors
	// or changes to an event that would affect the cached data?
	event.Fields.DeepUpdate(data.Clone())

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

func (p *addHostMetadata) cacheForFQDN(useFQDN bool) *hostMetadataCache {
	if useFQDN {
		return &p.caches[1]
	}
	return &p.caches[0]
}

func timestampExpired(timestamp time.Time, ttl time.Duration) bool {
	if ttl <= 0 {
		return true
	}
	return timestamp.Add(ttl).Before(time.Now())
}

// loadData update's the processor's associated host metadata
func (p *addHostMetadata) loadData(useFQDN bool) (mapstr.M, error) {
	cache := p.cacheForFQDN(useFQDN)
	cache.Lock()
	defer cache.Unlock()

	data := cache.data.Get()
	var err error
	if data == nil || timestampExpired(cache.lastUpdate, p.config.CacheTTL) {
		data, err = p.fetchData(useFQDN)
		if err == nil {
			cache.data.Set(data)
		}
		// Backwards compatibility (for now): cache timestamp is updated even if
		// the update fails (falls back on the last successful update, and avoids
		// blocking the pipeline when there are issues with the hostname).
		cache.lastUpdate = time.Now()
	}
	return data, err
}

func (p *addHostMetadata) fetchData(useFQDN bool) (mapstr.M, error) {
	h, err := p.hostInfoFactory()
	if err != nil {
		return nil, fmt.Errorf("error collecting host info: %w", err)
	}

	hInfo := h.Info()
	hostname := hInfo.Hostname
	if useFQDN {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		fqdn, err := h.FQDNWithContext(ctx)
		if err != nil {
			// FQDN lookup is "best effort". If it fails, we monitor the failure, fallback to
			// the OS-reported hostname, and move on.
			p.metrics.FQDNLookupFailed.Inc()
			p.logger.Warnf(
				"unable to lookup FQDN (failed attempt counter: %d): %s, using hostname = %s as FQDN",
				p.metrics.FQDNLookupFailed.Get(),
				err.Error(),
				hostname,
			)
		} else {
			hostname = fqdn
		}
	}

	data := host.MapHostInfo(hInfo, hostname)
	if p.config.NetInfoEnabled {
		// IP-address and MAC-address
		var ipList, hwList, err = util.GetNetInfo()
		if err != nil {
			p.logger.Infof("Error when getting network information %v", err)
		}

		if len(ipList) > 0 {
			if _, err := data.Put("host.ip", ipList); err != nil {
				return nil, fmt.Errorf("could not set host.ip: %w", err)
			}
		}
		if len(hwList) > 0 {
			if _, err := data.Put("host.mac", hwList); err != nil {
				return nil, fmt.Errorf("could not set host.mac: %w", err)
			}
		}
	}

	if p.config.Name != "" {
		if _, err := data.Put("host.name", p.config.Name); err != nil {
			return nil, fmt.Errorf("could not set host.name: %w", err)
		}
	}

	return data, nil
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
