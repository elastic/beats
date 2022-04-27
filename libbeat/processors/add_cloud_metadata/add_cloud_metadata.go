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

package add_cloud_metadata

import (
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/processors"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor"
	cfg "github.com/elastic/elastic-agent-libs/config"
)

const (
	// metadataHost is the IP that each of the cloud providers supported here
	// use for their metadata service.
	metadataHost = "169.254.169.254"
)

// init registers the add_cloud_metadata processor.
func init() {
	processors.RegisterPlugin("add_cloud_metadata", New)
	jsprocessor.RegisterPlugin("AddCloudMetadata", New)
}

type addCloudMetadata struct {
	initOnce sync.Once
	initData *initData
	metadata common.MapStr
	logger   *logp.Logger
}

type initData struct {
	fetchers  []metadataFetcher
	timeout   time.Duration
	tlsConfig *tlscommon.TLSConfig
	overwrite bool
}

// New constructs a new add_cloud_metadata processor.
func New(c *cfg.C) (processors.Processor, error) {
	config := defaultConfig()
	if err := c.Unpack(&config); err != nil {
		return nil, errors.Wrap(err, "failed to unpack add_cloud_metadata config")
	}

	tlsConfig, err := tlscommon.LoadTLSConfig(config.TLS)
	if err != nil {
		return nil, errors.Wrap(err, "TLS configuration load")
	}

	initProviders := selectProviders(config.Providers, cloudMetaProviders)
	fetchers, err := setupFetchers(initProviders, c)
	if err != nil {
		return nil, err
	}
	p := &addCloudMetadata{
		initData: &initData{
			fetchers:  fetchers,
			timeout:   config.Timeout,
			tlsConfig: tlsConfig,
			overwrite: config.Overwrite,
		},
		logger: logp.NewLogger("add_cloud_metadata"),
	}

	go p.init()
	return p, nil
}

func (r result) String() string {
	return fmt.Sprintf("result=[provider:%v, error=%v, metadata=%v]",
		r.provider, r.err, r.metadata)
}

func (p *addCloudMetadata) init() {
	p.initOnce.Do(func() {
		result := p.fetchMetadata()
		if result == nil {
			p.logger.Info("add_cloud_metadata: hosting provider type not detected.")
			return
		}
		p.metadata = result.metadata
		p.logger.Infof("add_cloud_metadata: hosting provider type detected as %v, metadata=%v",
			result.provider, result.metadata.String())
	})
}

func (p *addCloudMetadata) getMeta() common.MapStr {
	p.init()
	return p.metadata.Clone()
}

func (p *addCloudMetadata) Run(event *beat.Event) (*beat.Event, error) {
	meta := p.getMeta()
	if len(meta) == 0 {
		return event, nil
	}

	err := p.addMeta(event, meta)
	if err != nil {
		return nil, err
	}
	return event, err
}

func (p *addCloudMetadata) String() string {
	return "add_cloud_metadata=" + p.getMeta().String()
}

func (p *addCloudMetadata) addMeta(event *beat.Event, meta common.MapStr) error {
	for key, metaVal := range meta {
		// If key exists in event already and overwrite flag is set to false, this processor will not overwrite the
		// meta fields. For example aws module writes cloud.instance.* to events already, with overwrite=false,
		// add_cloud_metadata should not overwrite these fields with new values.
		if !p.initData.overwrite {
			v, _ := event.GetValue(key)
			if v != nil {
				continue
			}
		}
		_, err := event.PutValue(key, metaVal)
		if err != nil {
			return err
		}
	}
	return nil
}
