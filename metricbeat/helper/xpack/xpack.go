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

package xpack

import (
	"fmt"
	"time"
)

// Product supported by X-Pack Monitoring
type Product int

const (
	// Elasticsearch product
	Elasticsearch Product = iota

	// Kibana product
	Kibana

	// Logstash product
	Logstash

	// Beats product
	Beats
)

func (p Product) String() string {
	indexProductNames := []string{
		"es",
		"kibana",
		"logstash",
		"beats",
	}

	if int(p) < 0 || int(p) > len(indexProductNames) {
		panic("Unknown product")
	}

	return indexProductNames[p]
}

// MakeMonitoringIndexName method returns the name of the monitoring index for
// a given product { elasticsearch, kibana, logstash, beats }
func MakeMonitoringIndexName(product Product) string {
	today := time.Now().UTC().Format("2006.01.02")
	const version = "6"

	return fmt.Sprintf(".monitoring-%v-%v-mb-%v", product, version, today)
}
