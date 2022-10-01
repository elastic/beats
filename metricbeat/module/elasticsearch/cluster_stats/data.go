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

package cluster_stats

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"sort"
	"strings"

	"github.com/pkg/errors"

	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/helper"
	"github.com/elastic/beats/v7/metricbeat/helper/elastic"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var (
	schema = s.Schema{
		"status": c.Str("status"),
		"nodes": c.Dict("nodes", s.Schema{
			"versions": c.Ifc("versions"),
			"count":    c.Int("count.total"),
			"master":   c.Int("count.master"),
			"fs": c.Dict("fs", s.Schema{
				"total": s.Object{
					"bytes": c.Int("total_in_bytes"),
				},
				"available": s.Object{
					"bytes": c.Int("available_in_bytes"),
				},
			}),
			"jvm": c.Dict("jvm", s.Schema{
				"max_uptime": s.Object{
					"ms": c.Int("max_uptime_in_millis"),
				},
				"memory": c.Dict("mem", s.Schema{
					"heap": s.Object{
						"used": s.Object{
							"bytes": c.Int("heap_used_in_bytes"),
						},
						"max": s.Object{
							"bytes": c.Int("heap_max_in_bytes"),
						},
					},
				}),
			}),
		}),

		"indices": c.Dict("indices", s.Schema{
			"docs": c.Dict("docs", s.Schema{
				"total": c.Int("count"),
			}),
			"total": c.Int("count"),
			"shards": c.Dict("shards", s.Schema{
				"count":     c.Int("total"),
				"primaries": c.Int("primaries"),
			}),
			"store": c.Dict("store", s.Schema{
				"size": s.Object{"bytes": c.Int("size_in_bytes")},
			}),
			"fielddata": c.Dict("fielddata", s.Schema{
				"memory": s.Object{
					"bytes": c.Int("memory_size_in_bytes"),
				},
			}),
		}),
	}

	stackSchema = s.Schema{
		"xpack": c.Dict("xpack", s.Schema{
			"ccr": c.Dict("ccr", s.Schema{
				"enabled":   c.Bool("enabled"),
				"available": c.Bool("available"),
			}),
		}),
	}
)

func clusterNeedsTLSEnabled(license *elasticsearch.License, stackStats mapstr.M) (bool, error) {
	// TLS does not need to be enabled if license type is something other than trial
	if !license.IsOneOf("trial") {
		return false, nil
	}

	// TLS does not need to be enabled if security is not enabled
	value, err := stackStats.GetValue("security.enabled")
	if err != nil {
		return false, elastic.MakeErrorForMissingField("security.enabled", elastic.Elasticsearch)
	}

	isSecurityEnabled, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("security enabled flag is not a boolean")
	}

	if !isSecurityEnabled {
		return false, nil
	}

	// TLS does not need to be enabled if TLS is already enabled on the transport protocol
	value, err = stackStats.GetValue("security.ssl.transport.enabled")
	if err != nil {
		return false, elastic.MakeErrorForMissingField("security.ssl.transport.enabled", elastic.Elasticsearch)
	}

	isTLSAlreadyEnabled, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("transport protocol SSL enabled flag is not a boolean")
	}

	return !isTLSAlreadyEnabled, nil
}

// computeNodesHash computes a simple hash value that can be used to determine if the nodes listing has changed since the last report.
func computeNodesHash(clusterState mapstr.M) (int32, error) {
	value, err := clusterState.GetValue("nodes")
	if err != nil {
		return 0, elastic.MakeErrorForMissingField("nodes", elastic.Elasticsearch)
	}

	nodes, ok := value.(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("nodes is not a map")
	}

	var nodeEphemeralIDs []string
	for _, value := range nodes {
		nodeData, ok := value.(map[string]interface{})
		if !ok {
			return 0, fmt.Errorf("node data is not a map")
		}

		value, ok := nodeData["ephemeral_id"]
		if !ok {
			return 0, fmt.Errorf("node data does not contain ephemeral ID")
		}

		ephemeralID, ok := value.(string)
		if !ok {
			return 0, fmt.Errorf("node ephemeral ID is not a string")
		}

		nodeEphemeralIDs = append(nodeEphemeralIDs, ephemeralID)
	}

	sort.Strings(nodeEphemeralIDs)

	combinedNodeEphemeralIDs := strings.Join(nodeEphemeralIDs, "")
	return hash(combinedNodeEphemeralIDs), nil
}

func hash(s string) int32 {
	h := fnv.New32()
	h.Write([]byte(s))
	return int32(h.Sum32()) // This cast is needed because the ES mapping is for a 32-bit *signed* integer
}

func apmIndicesExist(clusterState mapstr.M) (bool, error) {
	value, err := clusterState.GetValue("routing_table.indices")
	if err != nil {
		return false, elastic.MakeErrorForMissingField("routing_table.indices", elastic.Elasticsearch)
	}

	indices, ok := value.(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("routing table indices is not a map")
	}

	for name := range indices {
		if strings.HasPrefix(name, "apm-") {
			return true, nil
		}
	}

	return false, nil
}

func getClusterMetadataSettings(httpClient *helper.HTTP) (mapstr.M, error) {
	// For security reasons we only get the display_name setting
	filterPaths := []string{"*.cluster.metadata.display_name"}
	clusterSettings, err := elasticsearch.GetClusterSettingsWithDefaults(httpClient, httpClient.GetURI(), filterPaths)
	if err != nil {
		return nil, errors.Wrap(err, "failure to get cluster settings")
	}

	clusterSettings, err = elasticsearch.MergeClusterSettings(clusterSettings)
	if err != nil {
		return nil, errors.Wrap(err, "failure to merge cluster settings")
	}

	return clusterSettings, nil
}

func eventMapping(r mb.ReporterV2, httpClient *helper.HTTP, info elasticsearch.Info, content []byte, isXpack bool) error {
	var data map[string]interface{}
	err := json.Unmarshal(content, &data)
	if err != nil {
		return errors.Wrap(err, "failure parsing Elasticsearch Cluster Stats API response")
	}

	clusterStats := mapstr.M(data)
	clusterStats.Delete("_nodes")

	license, err := elasticsearch.GetLicense(httpClient, httpClient.GetURI())
	if err != nil {
		return errors.Wrap(err, "failed to get license from Elasticsearch")
	}

	clusterStateMetrics := []string{"version", "master_node", "nodes", "routing_table"}
	clusterState, err := elasticsearch.GetClusterState(httpClient, httpClient.GetURI(), clusterStateMetrics)
	if err != nil {
		return errors.Wrap(err, "failed to get cluster state from Elasticsearch")
	}
	clusterState.Delete("cluster_name")

	clusterStateReduced := mapstr.M{}
	if err = elasticsearch.PassThruField("status", clusterStats, clusterStateReduced); err != nil {
		return errors.Wrap(err, "failed to pass through status field")
	}
	clusterStateReduced.Delete("status")

	if err = elasticsearch.PassThruField("master_node", clusterState, clusterStateReduced); err != nil {
		return errors.Wrap(err, "failed to pass through master_node field")
	}

	if err = elasticsearch.PassThruField("state_uuid", clusterState, clusterStateReduced); err != nil {
		return errors.Wrap(err, "failed to pass through state_uuid field")
	}

	if err = elasticsearch.PassThruField("nodes", clusterState, clusterStateReduced); err != nil {
		return errors.Wrap(err, "failed to pass through nodes field")
	}

	nodesHash, err := computeNodesHash(clusterState)
	if err != nil {
		return errors.Wrap(err, "failed to compute nodes hash")
	}
	clusterStateReduced.Put("nodes_hash", nodesHash)

	usage, err := elasticsearch.GetStackUsage(httpClient, httpClient.GetURI())
	if err != nil {
		return errors.Wrap(err, "failed to get stack usage from Elasticsearch")
	}

	clusterNeedsTLS, err := clusterNeedsTLSEnabled(license, usage)
	if err != nil {
		return errors.Wrap(err, "failed to determine if cluster needs TLS enabled")
	}

	l := license.ToMapStr()
	l["cluster_needs_tls"] = clusterNeedsTLS

	isAPMFound, err := apmIndicesExist(clusterState)
	if err != nil {
		return errors.Wrap(err, "failed to determine if APM indices exist")
	}
	delete(clusterState, "routing_table") // We don't want to index the routing table in monitoring indices

	stackStats := map[string]interface{}{
		"xpack": usage,
		"apm": map[string]interface{}{
			"found": isAPMFound,
		},
	}
	stackData, _ := stackSchema.Apply(stackStats)

	event := mb.Event{
		ModuleFields: mapstr.M{},
		RootFields:   mapstr.M{},
	}
	event.ModuleFields.Put("cluster.name", info.ClusterName)
	event.ModuleFields.Put("cluster.id", info.ClusterID)

	clusterSettings, err := getClusterMetadataSettings(httpClient)
	if err != nil {
		return err
	}
	if clusterSettings != nil {
		event.RootFields.Put("cluster_settings", clusterSettings)
	}

	metricSetFields, _ := schema.Apply(data)

	metricSetFields.Put("stack", stackData)
	metricSetFields.Put("license", l)
	metricSetFields.Put("state", clusterStateReduced)

	if err = elasticsearch.PassThruField("version", clusterState, event.ModuleFields); err != nil {
		return errors.Wrap(err, "failed to pass through version field")
	}

	event.MetricSetFields = metricSetFields

	// xpack.enabled in config using standalone metricbeat writes to `.monitoring` instead of `metricbeat-*`
	// When using Agent, the index name is overwritten anyways.
	if isXpack {
		index := elastic.MakeXPackMonitoringIndexName(elastic.Elasticsearch)
		event.Index = index
	}

	r.Event(event)

	return nil
}
