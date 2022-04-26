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

package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/helper/server"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type metricProcessor struct {
	paths       map[string]PathConfig
	defaultPath PathConfig
	sync.RWMutex
}

func NewMetricProcessor(paths []PathConfig, defaultPath PathConfig) *metricProcessor {
	pathMap := map[string]PathConfig{}
	for _, path := range paths {
		pathMap[path.Path] = path
	}

	return &metricProcessor{
		paths:       pathMap,
		defaultPath: defaultPath,
	}
}

func (m *metricProcessor) AddPath(path PathConfig) {
	m.Lock()
	m.paths[path.Path] = path
	m.Unlock()
}

func (m *metricProcessor) RemovePath(path PathConfig) {
	m.Lock()
	delete(m.paths, path.Path)
	m.Unlock()
}

func (p *metricProcessor) Process(event server.Event) (mapstr.M, error) {
	urlRaw, ok := event.GetMeta()["path"]
	if !ok {
		return nil, errors.New("Malformed HTTP event. Path missing.")
	}
	url, _ := urlRaw.(string)

	typeRaw, ok := event.GetMeta()["Content-Type"]
	if !ok {
		return nil, errors.New("Unable to get Content-Type of request")
	}
	contentType := typeRaw.(string)
	pathConf := p.findPath(url)

	bytesRaw, ok := event.GetEvent()[server.EventDataKey]
	if !ok {
		return nil, errors.New("Unable to retrieve response bytes")
	}

	bytes, _ := bytesRaw.([]byte)
	if len(bytes) == 0 {
		return nil, errors.New("Request has no data")
	}

	out := mapstr.M{}
	switch contentType {
	case "application/json":
		err := json.Unmarshal(bytes, &out)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New(fmt.Sprintf("Unsupported Content-Type: %s", contentType))
	}

	out[mb.NamespaceKey] = pathConf.Namespace
	if len(pathConf.Fields) != 0 {
		// Overwrite any keys that are present in the incoming payload
		common.MergeFields(out, pathConf.Fields, true)
	}
	return out, nil
}

func (p *metricProcessor) findPath(url string) *PathConfig {
	for path, conf := range p.paths {
		if strings.Index(url, path) == 0 {
			return &conf
		}
	}

	return &p.defaultPath
}
