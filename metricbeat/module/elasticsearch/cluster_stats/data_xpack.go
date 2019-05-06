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
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/helper/elastic"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/elasticsearch"
)

type clusterStatsLicense struct {
	*elasticsearch.License
	ClusterNeedsTLS bool `json:"cluster_needs_tls"`
}

func clusterNeedsTLSEnabled(license *elasticsearch.License, stackStats common.MapStr) (bool, error) {
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
func computeNodesHash(clusterState common.MapStr) (int32, error) {
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

func apmIndicesExist(clusterState common.MapStr) (bool, error) {
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

func getClusterMetadataSettings(m *MetricSet) (common.MapStr, error) {
	// For security reasons we only get the display_name setting
	filterPaths := []string{"*.cluster.metadata.display_name"}
	clusterSettings, err := elasticsearch.GetClusterSettingsWithDefaults(m.HTTP, m.HTTP.GetURI(), filterPaths)
	if err != nil {
		return nil, errors.Wrap(err, "failure to get cluster settings")
	}

	clusterSettings, err = elasticsearch.MergeClusterSettings(clusterSettings)
	if err != nil {
		return nil, errors.Wrap(err, "failure to merge cluster settings")
	}

	return clusterSettings, nil
}

func eventMappingXPack(r mb.ReporterV2, m *MetricSet, info elasticsearch.Info, content []byte) error {
	var data map[string]interface{}
	err := json.Unmarshal(content, &data)
	if err != nil {
		return errors.Wrap(err, "failure parsing Elasticsearch Cluster Stats API response")
	}

	clusterStats := common.MapStr(data)
	clusterStats.Delete("_nodes")

	value, err := clusterStats.GetValue("cluster_name")
	if err != nil {
		return elastic.MakeErrorForMissingField("cluster_name", elastic.Elasticsearch)
	}
	clusterName, ok := value.(string)
	if !ok {
		return fmt.Errorf("cluster name is not a string")
	}
	clusterStats.Delete("cluster_name")

	license, err := elasticsearch.GetLicense(m.HTTP, m.HTTP.GetURI())
	if err != nil {
		return errors.Wrap(err, "failed to get license from Elasticsearch")
	}

	clusterStateMetrics := []string{"version", "master_node", "nodes", "routing_table"}
	clusterState, err := elasticsearch.GetClusterState(m.HTTP, m.HTTP.GetURI(), clusterStateMetrics)
	if err != nil {
		return errors.Wrap(err, "failed to get cluster state from Elasticsearch")
	}
	clusterState.Delete("cluster_name")

	if err = elasticsearch.PassThruField("status", clusterStats, clusterState); err != nil {
		return errors.Wrap(err, "failed to pass through status field")
	}

	nodesHash, err := computeNodesHash(clusterState)
	if err != nil {
		return errors.Wrap(err, "failed to compute nodes hash")
	}
	clusterState.Put("nodes_hash", nodesHash)

	usage, err := elasticsearch.GetStackUsage(m.HTTP, m.HTTP.GetURI())
	if err != nil {
		return errors.Wrap(err, "failed to get stack usage from Elasticsearch")
	}

	clusterNeedsTLS, err := clusterNeedsTLSEnabled(license, usage)
	if err != nil {
		return errors.Wrap(err, "failed to determine if cluster needs TLS enabled")
	}

	l := clusterStatsLicense{license, clusterNeedsTLS}

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

	event := mb.Event{}
	event.RootFields = common.MapStr{
		"cluster_uuid":  info.ClusterID,
		"cluster_name":  clusterName,
		"timestamp":     common.Time(time.Now()),
		"interval_ms":   m.Module().Config().Period / time.Millisecond,
		"type":          "cluster_stats",
		"license":       l,
		"version":       info.Version.Number.String(),
		"cluster_stats": clusterStats,
		"cluster_state": clusterState,
		"stack_stats":   stackStats,
	}

	clusterSettings, err := getClusterMetadataSettings(m)
	if err != nil {
		return err
	}
	if clusterSettings != nil {
		event.RootFields.Put("cluster_settings", clusterSettings)
	}

	event.Index = elastic.MakeXPackMonitoringIndexName(elastic.Elasticsearch)
	r.Event(event)

	return nil
}
