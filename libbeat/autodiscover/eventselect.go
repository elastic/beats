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

package autodiscover

import (
	"fmt"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/common/bus"
)

type queryConfigFrom string

var defaultConfigQuery = queryConfigFrom("config")

// EventConfigQuery creates an EventConfigurer that tries to cast the given event
// field from from the buf event into a []*common.Config.
func EventConfigQuery(field string) EventConfigurer {
	if field == "" || field == "config" {
		return defaultConfigQuery
	}
	return queryConfigFrom(field)
}

// QueryConfig extract an array of *common.Config from bus.Event.
// The configurations are expected to be in the 'config' field.
func QueryConfig() EventConfigurer { return defaultConfigQuery }

func (q queryConfigFrom) EventFilter() []string { return []string{string(q)} }

func (q queryConfigFrom) CreateConfig(e bus.Event) ([]*common.Config, error) {
	fieldName := string(q)
	config, ok := e[fieldName].([]*common.Config)
	if !ok {
		return nil, fmt.Errorf("Event field '%v' does not contain a valid configuration object", fieldName)
	}
	return config, nil
}
