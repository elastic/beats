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

package jmx

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/menderesk/beats/v7/libbeat/common"
)

func TestEventMapper(t *testing.T) {
	absPath, err := filepath.Abs("./_meta/test")

	require.NotNil(t, absPath)
	require.NoError(t, err)

	jolokiaResponse, err := ioutil.ReadFile(absPath + "/jolokia_response.json")

	require.NoError(t, err)

	var mapping = AttributeMapping{
		attributeMappingKey{"java.lang:type=Runtime", "Uptime"}: Attribute{
			Attr: "Uptime", Field: "uptime"},
		attributeMappingKey{"java.lang:type=GarbageCollector,name=ConcurrentMarkSweep", "CollectionTime"}: Attribute{
			Attr: "CollectionTime", Field: "gc.cms_collection_time"},
		attributeMappingKey{"java.lang:type=GarbageCollector,name=ConcurrentMarkSweep", "CollectionCount"}: Attribute{
			Attr: "CollectionCount", Field: "gc.cms_collection_count"},
		attributeMappingKey{"java.lang:type=Memory", "HeapMemoryUsage"}: Attribute{
			Attr: "HeapMemoryUsage", Field: "memory.heap_usage"},
		attributeMappingKey{"java.lang:type=Memory", "NonHeapMemoryUsage"}: Attribute{
			Attr: "NonHEapMemoryUsage", Field: "memory.non_heap_usage"},
		attributeMappingKey{"org.springframework.boot:type=Endpoint,name=metricsEndpoint", "Metrics"}: Attribute{
			Attr: "Metrics", Field: "metrics"},
		attributeMappingKey{"Catalina:type=Server", "serverInfo"}: Attribute{
			Attr: "serverInfo", Field: "server_info"},
	}

	// Construct a new POST response event mapper
	eventMapper := NewJolokiaHTTPRequestFetcher("POST")

	// Map response to Metricbeat events
	events, err := eventMapper.EventMapping(jolokiaResponse, mapping)

	require.NoError(t, err)

	expected := []common.MapStr{
		{
			"uptime": float64(47283),
			"gc": common.MapStr{
				"cms_collection_time":  float64(53),
				"cms_collection_count": float64(1),
			},
			"memory": common.MapStr{
				"heap_usage": map[string]interface{}{
					"init":      float64(1073741824),
					"committed": float64(1037959168),
					"max":       float64(1037959168),
					"used":      float64(227420472),
				},
				"non_heap_usage": map[string]interface{}{
					"init":      float64(2555904),
					"committed": float64(53477376),
					"max":       float64(-1),
					"used":      float64(50519768),
				},
			},
			"metrics": map[string]interface{}{
				"atomikos_nbTransactions": float64(0),
				"classes":                 float64(18857),
				"classes_loaded":          float64(19127),
				"classes_unloaded":        float64(270),
			},
			"server_info": "Apache Tomcat/9.0.7",
		},
	}

	require.ElementsMatch(t, expected, events)
}

// TestEventGroupingMapper tests responses which are returned
// from a Jolokia POST request.
func TestEventGroupingMapper(t *testing.T) {
	absPath, err := filepath.Abs("./_meta/test")

	require.NotNil(t, absPath)
	require.NoError(t, err)

	jolokiaResponse, err := ioutil.ReadFile(absPath + "/jolokia_response.json")

	require.NoError(t, err)

	var mapping = AttributeMapping{
		attributeMappingKey{"java.lang:type=Runtime", "Uptime"}: Attribute{
			Attr: "Uptime", Field: "uptime"},
		attributeMappingKey{"java.lang:type=GarbageCollector,name=ConcurrentMarkSweep", "CollectionTime"}: Attribute{
			Attr: "CollectionTime", Field: "gc.cms_collection_time", Event: "gc"},
		attributeMappingKey{"java.lang:type=GarbageCollector,name=ConcurrentMarkSweep", "CollectionCount"}: Attribute{
			Attr: "CollectionCount", Field: "gc.cms_collection_count", Event: "gc"},
		attributeMappingKey{"java.lang:type=Memory", "HeapMemoryUsage"}: Attribute{
			Attr: "HeapMemoryUsage", Field: "memory.heap_usage", Event: "memory"},
		attributeMappingKey{"java.lang:type=Memory", "NonHeapMemoryUsage"}: Attribute{
			Attr: "NonHEapMemoryUsage", Field: "memory.non_heap_usage", Event: "memory"},
		attributeMappingKey{"org.springframework.boot:type=Endpoint,name=metricsEndpoint", "Metrics"}: Attribute{
			Attr: "Metrics", Field: "metrics"},
		attributeMappingKey{"Catalina:type=Server", "serverInfo"}: Attribute{
			Attr: "serverInfo", Field: "server_info"},
	}

	// Construct a new POST response event mapper
	eventMapper := NewJolokiaHTTPRequestFetcher("POST")

	// Map response to Metricbeat events
	events, err := eventMapper.EventMapping(jolokiaResponse, mapping)

	require.NoError(t, err)

	expected := []common.MapStr{
		{
			"uptime": float64(47283),
			"metrics": map[string]interface{}{
				"atomikos_nbTransactions": float64(0),
				"classes":                 float64(18857),
				"classes_loaded":          float64(19127),
				"classes_unloaded":        float64(270),
			},
			"server_info": "Apache Tomcat/9.0.7",
		},
		{
			"gc": common.MapStr{
				"cms_collection_time":  float64(53),
				"cms_collection_count": float64(1),
			},
		},
		{
			"memory": common.MapStr{
				"heap_usage": map[string]interface{}{
					"init":      float64(1073741824),
					"committed": float64(1037959168),
					"max":       float64(1037959168),
					"used":      float64(227420472),
				},
				"non_heap_usage": map[string]interface{}{
					"init":      float64(2555904),
					"committed": float64(53477376),
					"max":       float64(-1),
					"used":      float64(50519768),
				},
			},
		},
	}

	require.ElementsMatch(t, expected, events)
}

// TestEventGroupingMapperGetRequest tests responses which are returned
// from a Jolokia GET request. The difference from POST responses is that
// GET method returns a single Entry, whereas POST method returns an array
// of Entry objects
func TestEventGroupingMapperGetRequest(t *testing.T) {
	absPath, err := filepath.Abs("./_meta/test")

	require.NotNil(t, absPath)
	require.NoError(t, err)

	jolokiaResponse, err := ioutil.ReadFile(absPath + "/jolokia_get_response.json")

	require.NoError(t, err)

	var mapping = AttributeMapping{
		attributeMappingKey{"java.lang:type=Memory", "HeapMemoryUsage"}: Attribute{
			Attr: "HeapMemoryUsage", Field: "memory.heap_usage", Event: "memory"},
		attributeMappingKey{"java.lang:type=Memory", "NonHeapMemoryUsage"}: Attribute{
			Attr: "NonHEapMemoryUsage", Field: "memory.non_heap_usage", Event: "memory"},
	}

	// Construct a new GET response event mapper
	eventMapper := NewJolokiaHTTPRequestFetcher("GET")

	// Map response to Metricbeat events
	events, err := eventMapper.EventMapping(jolokiaResponse, mapping)

	require.NoError(t, err)

	expected := []common.MapStr{
		{
			"memory": common.MapStr{
				"heap_usage": map[string]interface{}{
					"init":      float64(1073741824),
					"committed": float64(1037959168),
					"max":       float64(1037959168),
					"used":      float64(227420472),
				},
				"non_heap_usage": map[string]interface{}{
					"init":      float64(2555904),
					"committed": float64(53477376),
					"max":       float64(-1),
					"used":      float64(50519768),
				},
			},
		},
	}

	require.ElementsMatch(t, expected, events)
}

// TestEventGroupingMapperGetRequestUptime tests responses which are returned
// from a Jolokia GET request and only has one uptime runtime value.
func TestEventGroupingMapperGetRequestUptime(t *testing.T) {
	absPath, err := filepath.Abs("./_meta/test")

	require.NotNil(t, absPath)
	require.NoError(t, err)

	jolokiaResponse, err := ioutil.ReadFile(absPath + "/jolokia_get_response_uptime.json")

	require.NoError(t, err)

	var mapping = AttributeMapping{
		attributeMappingKey{"java.lang:type=Runtime", "Uptime"}: Attribute{
			Field: "runtime.uptime", Event: "runtime"},
	}

	// Construct a new GET response event mapper
	eventMapper := NewJolokiaHTTPRequestFetcher("GET")

	// Map response to Metricbeat events
	events, err := eventMapper.EventMapping(jolokiaResponse, mapping)

	require.NoError(t, err)

	expected := []common.MapStr{
		{
			"runtime": common.MapStr{
				"uptime": float64(88622),
			},
		},
	}

	require.ElementsMatch(t, expected, events)
}

func TestEventMapperWithWildcard(t *testing.T) {
	absPath, err := filepath.Abs("./_meta/test")

	require.NotNil(t, absPath)
	require.NoError(t, err)

	jolokiaResponse, err := ioutil.ReadFile(absPath + "/jolokia_response_wildcard.json")

	require.NoError(t, err)

	var mapping = AttributeMapping{
		attributeMappingKey{"Catalina:name=*,type=ThreadPool", "port"}: Attribute{
			Attr: "port", Field: "port"},
		attributeMappingKey{"Catalina:name=*,type=ThreadPool", "maxConnections"}: Attribute{
			Attr: "maxConnections", Field: "max_connections"},
	}

	// Construct a new POST response event mapper
	eventMapper := NewJolokiaHTTPRequestFetcher("POST")

	// Map response to Metricbeat events
	events, err := eventMapper.EventMapping(jolokiaResponse, mapping)
	require.NoError(t, err)
	require.Equal(t, 2, len(events))

	expected := []common.MapStr{
		{
			"mbean":           "Catalina:name=\"http-bio-8080\",type=ThreadPool",
			"max_connections": float64(200),
			"port":            float64(8080),
		},
		{
			"mbean":           "Catalina:name=\"ajp-bio-8009\",type=ThreadPool",
			"max_connections": float64(200),
			"port":            float64(8009),
		},
	}

	require.ElementsMatch(t, expected, events)
}

func TestEventGroupingMapperWithWildcard(t *testing.T) {
	absPath, err := filepath.Abs("./_meta/test")

	require.NotNil(t, absPath)
	require.NoError(t, err)

	jolokiaResponse, err := ioutil.ReadFile(absPath + "/jolokia_response_wildcard.json")

	require.NoError(t, err)

	var mapping = AttributeMapping{
		attributeMappingKey{"Catalina:name=*,type=ThreadPool", "port"}: Attribute{
			Attr: "port", Field: "port", Event: "port"},
		attributeMappingKey{"Catalina:name=*,type=ThreadPool", "maxConnections"}: Attribute{
			Attr: "maxConnections", Field: "max_connections", Event: "network"},
	}

	// Construct a new POST response event mapper
	eventMapper := NewJolokiaHTTPRequestFetcher("POST")

	// Map response to Metricbeat events
	events, err := eventMapper.EventMapping(jolokiaResponse, mapping)
	require.NoError(t, err)
	require.Equal(t, 4, len(events))

	expected := []common.MapStr{
		{
			"mbean": "Catalina:name=\"http-bio-8080\",type=ThreadPool",
			"port":  float64(8080),
		},
		{
			"mbean":           "Catalina:name=\"http-bio-8080\",type=ThreadPool",
			"max_connections": float64(200),
		},
		{
			"mbean": "Catalina:name=\"ajp-bio-8009\",type=ThreadPool",
			"port":  float64(8009),
		},
		{
			"mbean":           "Catalina:name=\"ajp-bio-8009\",type=ThreadPool",
			"max_connections": float64(200),
		},
	}

	require.ElementsMatch(t, expected, events)
}
