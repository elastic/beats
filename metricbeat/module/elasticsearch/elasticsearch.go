package elasticsearch

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/elastic/beats/metricbeat/helper"
)

// Global clusterIdCache. Assumption is that the same node id never can belong to a different cluster id
var clusterIDCache = map[string]string{}

// Info construct contains the data from the Elasticsearch / endpoint
type Info struct {
	ClusterName string `json:"cluster_name"`
	ClusterID   string `json:"cluster_uuid"`
}

// GetClusterID fetches cluster id for given nodeID
func GetClusterID(http *helper.HTTP, uri string, nodeID string) (string, error) {
	// Check if cluster id already cached. If yes, return it.
	if clusterID, ok := clusterIDCache[nodeID]; ok {
		return clusterID, nil
	}

	// Makes sure the http uri is reset to its inital value
	defer http.SetURI(uri)

	info, err := GetInfo(http, uri)
	if err != nil {
		return "", err
	}

	clusterIDCache[nodeID] = info.ClusterID
	return info.ClusterID, nil
}

// IsMaster checks if the given node host is a master node
//
// The detection of the master is done in two steps:
// * Fetch node name from /_nodes/_local/name
// * Fetch current master name from cluster state /_cluster/state/master_node
//
// The two names are compared
func IsMaster(http *helper.HTTP, uri string) (bool, error) {
	// Makes sure the http uri is reset to its inital value
	defer http.SetURI(uri)

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
	content, err := fetchPath(http, uri, "/_nodes/_local/nodes")
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
	content, err := fetchPath(http, uri, "_cluster/state/master_node")
	if err != nil {
		return "", err
	}

	clusterStruct := struct {
		MasterNode string `json:"master_node"`
	}{}

	json.Unmarshal(content, &clusterStruct)

	return clusterStruct.MasterNode, nil
}

// GetInfo returns the data for the Elasticsearch / endpoint
func GetInfo(http *helper.HTTP, uri string) (*Info, error) {

	// Parses the uri to replace the path
	u, _ := url.Parse(uri)
	u.Path = ""

	// Http helper includes the HostData with username and password
	http.SetURI(u.String())
	content, err := http.FetchContent()
	if err != nil {
		return nil, err
	}

	info := &Info{}
	json.Unmarshal(content, info)

	return info, nil
}

func fetchPath(http *helper.HTTP, uri, path string) ([]byte, error) {
	// Parses the uri to replace the path
	u, _ := url.Parse(uri)
	u.Path = path

	// Http helper includes the HostData with username and password
	http.SetURI(u.String())
	return http.FetchContent()
}
