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

package elasticsearch

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/elastic/beats/metricbeat/mb"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/helper/elastic"
)

func init() {
	// Register the ModuleFactory function for this module.
	if err := mb.Registry.AddModule(ModuleName, NewModule); err != nil {
		panic(err)
	}
}

// NewModule creates a new module after performing validation.
func NewModule(base mb.BaseModule) (mb.Module, error) {
	if err := validateXPackMetricsets(base); err != nil {
		return nil, err
	}

	return &base, nil
}

// Validate that correct metricsets have been specified if xpack.enabled = true.
func validateXPackMetricsets(base mb.BaseModule) error {
	config := struct {
		Metricsets   []string `config:"metricsets"`
		XPackEnabled bool     `config:"xpack.enabled"`
	}{}
	if err := base.UnpackConfig(&config); err != nil {
		return err
	}

	// Nothing to validate if xpack.enabled != true
	if !config.XPackEnabled {
		return nil
	}

	expectedXPackMetricsets := []string{
		"ccr",
		"cluster_stats",
		"index",
		"index_recovery",
		"index_summary",
		"ml_job",
		"node_stats",
		"shard",
	}

	if !common.MakeStringSet(config.Metricsets...).Equals(common.MakeStringSet(expectedXPackMetricsets...)) {
		return errors.Errorf("The %v module with xpack.enabled: true must have metricsets: %v", ModuleName, expectedXPackMetricsets)
	}

	return nil
}

// CCRStatsAPIAvailableVersion is the version of Elasticsearch since when the CCR stats API is available.
var CCRStatsAPIAvailableVersion = common.MustNewVersion("6.5.0")

// Global clusterIdCache. Assumption is that the same node id never can belong to a different cluster id.
var clusterIDCache = map[string]string{}

// ModuleName is the name of this module.
const ModuleName = "elasticsearch"

// Info construct contains the data from the Elasticsearch / endpoint
type Info struct {
	ClusterName string `json:"cluster_name"`
	ClusterID   string `json:"cluster_uuid"`
	Version     struct {
		Number *common.Version `json:"number"`
	} `json:"version"`
}

// NodeInfo struct cotains data about the node.
type NodeInfo struct {
	Host             string `json:"host"`
	TransportAddress string `json:"transport_address"`
	IP               string `json:"ip"`
	Name             string `json:"name"`
	ID               string
}

// License contains data about the Elasticsearch license
type License struct {
	Status             string     `json:"status"`
	ID                 string     `json:"uid"`
	Type               string     `json:"type"`
	IssueDate          *time.Time `json:"issue_date"`
	IssueDateInMillis  int        `json:"issue_date_in_millis"`
	ExpiryDate         *time.Time `json:"expiry_date,omitempty"`
	ExpiryDateInMillis int        `json:"expiry_date_in_millis,omitempty"`
	MaxNodes           int        `json:"max_nodes"`
	IssuedTo           string     `json:"issued_to"`
	Issuer             string     `json:"issuer"`
	StartDateInMillis  int        `json:"start_date_in_millis"`
}

type licenseWrapper struct {
	License License `json:"license"`
}

// GetClusterID fetches cluster id for given nodeID.
func GetClusterID(http *helper.HTTP, uri string, nodeID string) (string, error) {
	// Check if cluster id already cached. If yes, return it.
	if clusterID, ok := clusterIDCache[nodeID]; ok {
		return clusterID, nil
	}

	info, err := GetInfo(http, uri)
	if err != nil {
		return "", err
	}

	clusterIDCache[nodeID] = info.ClusterID
	return info.ClusterID, nil
}

// IsMaster checks if the given node host is a master node.
//
// The detection of the master is done in two steps:
// * Fetch node name from /_nodes/_local/name
// * Fetch current master name from cluster state /_cluster/state/master_node
//
// The two names are compared
func IsMaster(http *helper.HTTP, uri string) (bool, error) {

	node, err := getNodeName(http, uri)
	if err != nil {
		return false, err
	}

	master, err := getMasterName(http, uri)
	if err != nil {
		return false, err
	}

	return master == node, nil
}

func getNodeName(http *helper.HTTP, uri string) (string, error) {
	content, err := fetchPath(http, uri, "/_nodes/_local/nodes", "")
	if err != nil {
		return "", err
	}

	nodesStruct := struct {
		Nodes map[string]interface{} `json:"nodes"`
	}{}

	json.Unmarshal(content, &nodesStruct)

	// _local will only fetch one node info. First entry is node name
	for k := range nodesStruct.Nodes {
		return k, nil
	}
	return "", fmt.Errorf("No local node found")
}

func getMasterName(http *helper.HTTP, uri string) (string, error) {
	// TODO: evaluate on why when run with ?local=true request does not contain master_node field
	content, err := fetchPath(http, uri, "_cluster/state/master_node", "")
	if err != nil {
		return "", err
	}

	clusterStruct := struct {
		MasterNode string `json:"master_node"`
	}{}

	json.Unmarshal(content, &clusterStruct)

	return clusterStruct.MasterNode, nil
}

// GetInfo returns the data for the Elasticsearch / endpoint.
func GetInfo(http *helper.HTTP, uri string) (*Info, error) {

	content, err := fetchPath(http, uri, "/", "")
	if err != nil {
		return nil, err
	}

	info := &Info{}
	err = json.Unmarshal(content, &info)
	if err != nil {
		return nil, err
	}

	return info, nil
}

func fetchPath(http *helper.HTTP, uri, path string, query string) ([]byte, error) {
	defer http.SetURI(uri)

	// Parses the uri to replace the path
	u, _ := url.Parse(uri)
	u.Path = path
	u.RawQuery = query

	// Http helper includes the HostData with username and password
	http.SetURI(u.String())
	return http.FetchContent()
}

// GetNodeInfo returns the node information.
func GetNodeInfo(http *helper.HTTP, uri string, nodeID string) (*NodeInfo, error) {

	content, err := fetchPath(http, uri, "/_nodes/_local/nodes", "")
	if err != nil {
		return nil, err
	}

	nodesStruct := struct {
		Nodes map[string]*NodeInfo `json:"nodes"`
	}{}

	json.Unmarshal(content, &nodesStruct)

	// _local will only fetch one node info. First entry is node name
	for k, v := range nodesStruct.Nodes {
		// In case the nodeID is empty, first node info will be returned
		if k == nodeID || nodeID == "" {
			v.ID = k
			return v, nil
		}
	}
	return nil, fmt.Errorf("no node matched id %s", nodeID)
}

// GetLicense returns license information. Since we don't expect license information
// to change frequently, the information is cached for 1 minute to avoid
// hitting Elasticsearch frequently.
func GetLicense(http *helper.HTTP, resetURI string) (*License, error) {
	// First, check the cache
	license := licenseCache.get()

	// License found in cache, return it
	if license != nil {
		return license, nil
	}

	// License not found in cache, fetch it from Elasticsearch
	info, err := GetInfo(http, resetURI)
	if err != nil {
		return nil, err
	}
	var licensePath string
	if info.Version.Number.Major < 7 {
		licensePath = "_xpack/license"
	} else {
		licensePath = "_license"
	}

	content, err := fetchPath(http, resetURI, licensePath, "")
	if err != nil {
		return nil, err
	}

	var data licenseWrapper
	err = json.Unmarshal(content, &data)
	if err != nil {
		return nil, err
	}

	// Cache license for a minute
	license = &data.License
	licenseCache.set(license, time.Minute)

	return license, nil
}

// GetClusterState returns cluster state information.
func GetClusterState(http *helper.HTTP, resetURI string, metrics []string) (common.MapStr, error) {
	clusterStateURI := "_cluster/state"
	if metrics != nil && len(metrics) > 0 {
		clusterStateURI += "/" + strings.Join(metrics, ",")
	}

	content, err := fetchPath(http, resetURI, clusterStateURI, "")
	if err != nil {
		return nil, err
	}

	var clusterState map[string]interface{}
	err = json.Unmarshal(content, &clusterState)
	return clusterState, err
}

// GetClusterSettingsWithDefaults returns cluster settings.
func GetClusterSettingsWithDefaults(http *helper.HTTP, resetURI string, filterPaths []string) (common.MapStr, error) {
	return GetClusterSettings(http, resetURI, true, filterPaths)
}

// GetClusterSettings returns cluster settings
func GetClusterSettings(http *helper.HTTP, resetURI string, includeDefaults bool, filterPaths []string) (common.MapStr, error) {
	clusterSettingsURI := "_cluster/settings"
	var queryParams []string
	if includeDefaults {
		queryParams = append(queryParams, "include_defaults=true")
	}

	if filterPaths != nil && len(filterPaths) > 0 {
		filterPathQueryParam := "filter_path=" + strings.Join(filterPaths, ",")
		queryParams = append(queryParams, filterPathQueryParam)
	}

	queryString := strings.Join(queryParams, "&")

	content, err := fetchPath(http, resetURI, clusterSettingsURI, queryString)
	if err != nil {
		return nil, err
	}

	var clusterSettings map[string]interface{}
	err = json.Unmarshal(content, &clusterSettings)
	return clusterSettings, err
}

// GetStackUsage returns stack usage information.
func GetStackUsage(http *helper.HTTP, resetURI string) (common.MapStr, error) {
	content, err := fetchPath(http, resetURI, "_xpack/usage", "")
	if err != nil {
		return nil, err
	}

	var stackUsage map[string]interface{}
	err = json.Unmarshal(content, &stackUsage)
	return stackUsage, err
}

// IsMLockAllEnabled returns if the given Elasticsearch node has mlockall enabled
func IsMLockAllEnabled(http *helper.HTTP, resetURI, nodeID string) (bool, error) {
	content, err := fetchPath(http, resetURI, "_nodes/"+nodeID, "filter_path=nodes.*.process.mlockall")
	if err != nil {
		return false, err
	}

	var response map[string]map[string]map[string]map[string]bool
	err = json.Unmarshal(content, &response)
	if err != nil {
		return false, err
	}

	for _, nodeInfo := range response["nodes"] {
		mlockall := nodeInfo["process"]["mlockall"]
		return mlockall, nil
	}

	return false, fmt.Errorf("could not determine if mlockall is enabled on node ID = %v", nodeID)
}

// PassThruField copies the field at the given path from the given source data object into
// the same path in the given target data object.
func PassThruField(fieldPath string, sourceData, targetData common.MapStr) error {
	fieldValue, err := sourceData.GetValue(fieldPath)
	if err != nil {
		return elastic.MakeErrorForMissingField(fieldPath, elastic.Elasticsearch)
	}

	targetData.Put(fieldPath, fieldValue)
	return nil
}

// MergeClusterSettings merges cluster settings in the correct precedence order
func MergeClusterSettings(clusterSettings common.MapStr) (common.MapStr, error) {
	transientSettings, err := getSettingGroup(clusterSettings, "transient")
	if err != nil {
		return nil, err
	}

	persistentSettings, err := getSettingGroup(clusterSettings, "persistent")
	if err != nil {
		return nil, err
	}

	settings, err := getSettingGroup(clusterSettings, "default")
	if err != nil {
		return nil, err
	}

	// Transient settings override persistent settings which override default settings
	if settings == nil {
		settings = persistentSettings
	}

	if settings == nil {
		settings = transientSettings
	}

	if settings == nil {
		return nil, nil
	}

	if persistentSettings != nil {
		settings.DeepUpdate(persistentSettings)
	}

	if transientSettings != nil {
		settings.DeepUpdate(transientSettings)
	}

	return settings, nil
}

// Global cache for license information. Assumption is that license information changes infrequently.
var licenseCache = &_licenseCache{}

type _licenseCache struct {
	sync.RWMutex
	license  *License
	cachedOn time.Time
	ttl      time.Duration
}

func (c *_licenseCache) get() *License {
	c.Lock()
	defer c.Unlock()

	if time.Since(c.cachedOn) > c.ttl {
		// We are past the TTL, so invalidate cache
		c.license = nil
	}

	return c.license
}

func (c *_licenseCache) set(license *License, ttl time.Duration) {
	c.Lock()
	defer c.Unlock()

	c.license = license
	c.ttl = ttl
	c.cachedOn = time.Now()
}

// IsOneOf returns whether the license is one of the specified candidate licenses
func (l *License) IsOneOf(candidateLicenses ...string) bool {
	t := l.Type

	for _, candidateLicense := range candidateLicenses {
		if candidateLicense == t {
			return true
		}
	}

	return false
}

func getSettingGroup(allSettings common.MapStr, groupKey string) (common.MapStr, error) {
	hasSettingGroup, err := allSettings.HasKey(groupKey)
	if err != nil {
		return nil, errors.Wrap(err, "failure to determine if "+groupKey+" settings exist")
	}

	if !hasSettingGroup {
		return nil, nil
	}

	settings, err := allSettings.GetValue(groupKey)
	if err != nil {
		return nil, errors.Wrap(err, "failure to extract "+groupKey+" settings")
	}

	v, ok := settings.(map[string]interface{})
	if !ok {
		return nil, errors.Wrap(err, groupKey+" settings are not a map")
	}

	return common.MapStr(v), nil
}
