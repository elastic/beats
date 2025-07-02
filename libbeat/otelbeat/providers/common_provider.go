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

package providers

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"go.opentelemetry.io/collector/confmap"
)

var schemeMap = map[string]string{
	"fb": "filebeatreceiver",
	"mb": "metricbeatreceiver",
}

func LoadConfig(uri string, schemeName string) (*confmap.Retrieved, error) {
	if !strings.HasPrefix(uri, schemeName+":") {
		return nil, fmt.Errorf("%q uri is not supported by %q provider", uri, schemeName)
	}

	// Load beat config file
	cfg, err := cfgfile.Load(filepath.Clean(uri[len(schemeName)+1:]), nil)
	if err != nil {
		return nil, err
	}

	var receiverMap map[string]any
	err = cfg.Unpack(&receiverMap)
	if err != nil {
		return nil, err
	}

	receiverName := schemeMap[schemeName]
	// beat specific configuration is defined here
	cfgMap := map[string]any{
		"receivers": map[string]any{
			receiverName: receiverMap,
		},
		"service": map[string]any{
			"pipelines": map[string]any{
				"logs": map[string]any{
					"receivers": []string{receiverName},
				},
			},
		},
	}

	return confmap.NewRetrieved(cfgMap)
}
