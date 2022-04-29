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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/helper"
	"github.com/elastic/beats/v7/metricbeat/helper/elastic"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func init() {
	// Register the ModuleFactory function for this module.
	if err := mb.Registry.AddModule(ModuleName, NewModule); err != nil {
		panic(err)
	}
}

// NewModule creates a new module.
func NewModule(base mb.BaseModule) (mb.Module, error) {
	xpackEnabledMetricSets := []string{
		"ccr",
		"enrich",
		"cluster_stats",
		"index",
		"index_recovery",
		"index_summary",
		"ml_job",
		"node_stats",
		"shard",
	}
	return elastic.NewModule(&base, xpackEnabledMetricSets, logp.NewLogger(ModuleName))
}

var (
	// CCRStatsAPIAvailableVersion is the version of Elasticsearch since when the CCR stats API is available.
	CCRStatsAPIAvailableVersion = common.MustNewVersion("6.5.0")

	// EnrichStatsAPIAvailableVersion is the version of Elasticsearch since when the Enrich stats API is available.
	EnrichStatsAPIAvailableVersion = common.MustNewVersion("7.5.0")

	// BulkStatsAvailableVersion is the version since when bulk indexing stats are available
	BulkStatsAvailableVersion = common.MustNewVersion("8.0.0")

	//ExpandWildcardsHiddenAvailableVersion is the version since when the "expand_wildcards" query parameter to
	// the Indices Stats API can accept "hidden" as a value.
	ExpandWildcardsHiddenAvailableVersion = common.MustNewVersion("7.7.0")

	// Global clusterIdCache. Assumption is that the same node id never can belong to a different cluster id.
	clusterIDCache = map[string]string{}
)

// ModuleName is the name of this module.
const ModuleName = "elasticsearch"

// Info construct contains the data from the Elasticsearch / endpoint
type Info struct {
	ClusterName string  `json:"cluster_name"`
	ClusterID   string  `json:"cluster_uuid"`
	Version     Version `json:"version"`
	Name        string  `json:"name"`
}

// Version contains the semver formatted version of ES
type Version struct {
	Number *common.Version `json:"number"`
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
	MaxNodes           int        `json:"max_nodes,omitempty"`
	MaxResourceUnits   int        `json:"max_resource_units,omitempty"`
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

// isMaster checks if the given node host is a master node.
//
// The detection of the master is done in two steps:
// * Fetch node name from /_nodes/_local/name
// * Fetch current master name from cluster state /_cluster/state/master_node
//
// The two names are compared
func isMaster(http *helper.HTTP, uri string) (bool, error) {

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
	content, err := fetchPath(http, resetURI, "_license", "")
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
func GetClusterState(http *helper.HTTP, resetURI string, metrics []string) (mapstr.M, error) {
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
func GetClusterSettingsWithDefaults(http *helper.HTTP, resetURI string, filterPaths []string) (mapstr.M, error) {
	return GetClusterSettings(http, resetURI, true, filterPaths)
}

// GetClusterSettings returns cluster settings
func GetClusterSettings(http *helper.HTTP, resetURI string, includeDefaults bool, filterPaths []string) (mapstr.M, error) {
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
func GetStackUsage(http *helper.HTTP, resetURI string) (map[string]interface{}, error) {
	content, err := fetchPath(http, resetURI, "_xpack/usage", "")
	if err != nil {
		return nil, err
	}

	var stackUsage map[string]interface{}
	err = json.Unmarshal(content, &stackUsage)
	return stackUsage, err
}

type XPack struct {
	Features struct {
		CCR struct {
			Enabled bool `json:"enabled"`
		} `json:"CCR"`
	} `json:"features"`
}

// GetXPack returns information about xpack features.
func GetXPack(http *helper.HTTP, resetURI string) (XPack, error) {
	content, err := fetchPath(http, resetURI, "_xpack", "")

	if err != nil {
		return XPack{}, err
	}

	var xpack XPack
	err = json.Unmarshal(content, &xpack)
	return xpack, err
}

type boolStr bool

func (b *boolStr) UnmarshalJSON(raw []byte) error {
	var bs string
	err := json.Unmarshal(raw, &bs)
	if err != nil {
		return err
	}

	bv, err := strconv.ParseBool(bs)
	if err != nil {
		return err
	}

	*b = boolStr(bv)
	return nil
}

type IndexSettings struct {
	Hidden bool
}

// GetIndicesSettings returns a map of index names to their settings.
// Note that as of now it is optimized to fetch only the "hidden" index setting to keep the memory
// footprint of this function call as low as possible.
func GetIndicesSettings(http *helper.HTTP, resetURI string) (map[string]IndexSettings, error) {
	content, err := fetchPath(http, resetURI, "*/_settings", "filter_path=*.settings.index.hidden&expand_wildcards=all")

	if err != nil {
		return nil, errors.Wrap(err, "could not fetch indices settings")
	}

	var resp map[string]struct {
		Settings struct {
			Index struct {
				Hidden boolStr `json:"hidden"`
			} `json:"index"`
		} `json:"settings"`
	}

	err = json.Unmarshal(content, &resp)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse indices settings response")
	}

	ret := make(map[string]IndexSettings, len(resp))
	for index, settings := range resp {
		ret[index] = IndexSettings{
			Hidden: bool(settings.Settings.Index.Hidden),
		}
	}

	return ret, nil
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

// GetMasterNodeID returns the ID of the Elasticsearch cluster's master node
func GetMasterNodeID(http *helper.HTTP, resetURI string) (string, error) {
	content, err := fetchPath(http, resetURI, "_nodes/_master", "filter_path=nodes.*.name")
	if err != nil {
		return "", err
	}

	var response struct {
		Nodes map[string]interface{} `json:"nodes"`
	}

	if err := json.Unmarshal(content, &response); err != nil {
		return "", err
	}

	for nodeID, _ := range response.Nodes {
		return nodeID, nil
	}

	return "", errors.New("could not determine master node ID")
}

// PassThruField copies the field at the given path from the given source data object into
// the same path in the given target data object.
func PassThruField(fieldPath string, sourceData, targetData mapstr.M) error {
	fieldValue, err := sourceData.GetValue(fieldPath)
	if err != nil {
		return elastic.MakeErrorForMissingField(fieldPath, elastic.Elasticsearch)
	}

	targetData.Put(fieldPath, fieldValue)
	return nil
}

// MergeClusterSettings merges cluster settings in the correct precedence order
func MergeClusterSettings(clusterSettings mapstr.M) (mapstr.M, error) {
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

var (
	// Global cache for license information. Assumption is that license information changes infrequently.
	licenseCache = &_licenseCache{}

	// LicenseCacheEnabled controls whether license caching is enabled or not. Intended for test use.
	LicenseCacheEnabled = true
)

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
	if !LicenseCacheEnabled {
		return
	}

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

// ToMapStr converts the license to a mapstr.M. This is necessary
// for proper marshaling of the data before it's sent over the wire. In
// particular it ensures that ms-since-epoch values are marshaled as longs
// and not floats in scientific notation as Elasticsearch does not like that.
func (l *License) ToMapStr() mapstr.M {
	m := mapstr.M{
		"status":               l.Status,
		"uid":                  l.ID,
		"type":                 l.Type,
		"issue_date":           l.IssueDate,
		"issue_date_in_millis": l.IssueDateInMillis,
		"expiry_date":          l.ExpiryDate,
		"issued_to":            l.IssuedTo,
		"issuer":               l.Issuer,
		"start_date_in_millis": l.StartDateInMillis,
	}

	if l.ExpiryDateInMillis != 0 {
		// We don't want to record a 0 expiry date as this means the license has expired
		// in the Stack Monitoring UI
		m["expiry_date_in_millis"] = l.ExpiryDateInMillis
	}

	// Enterprise licenses have max_resource_units. All other licenses have
	// max_nodes.
	if l.MaxNodes != 0 {
		m["max_nodes"] = l.MaxNodes
	}

	if l.MaxResourceUnits != 0 {
		m["max_resource_units"] = l.MaxResourceUnits
	}

	return m
}

func getSettingGroup(allSettings mapstr.M, groupKey string) (mapstr.M, error) {
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

	return mapstr.M(v), nil
}
