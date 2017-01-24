package nifi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
)

func init() {
	// Register the ModuleFactory function for the "mongodb" module.
	if err := mb.Registry.AddModule("nifi", NewModule); err != nil {
		panic(err)
	}
}

// NewModule creates a new mb.Module instance and validates that at least one host has been
// specified
func NewModule(base mb.BaseModule) (mb.Module, error) {
	// Validate that at least one host has been specified.
	config := struct {
		Hosts []string `config:"hosts"    validate:"nonzero,required"`
	}{}
	if err := base.UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &base, nil
}

// IsCluster ...
func IsCluster(host string, client *http.Client) bool {
	url := fmt.Sprintf("http://%s/nifi-api/controller/cluster", host)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logp.Err(err.Error())
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		logp.Err(err.Error())
		return false
	}

	defer resp.Body.Close()

	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return true
	}
	return false
}

// ClusterResponse ...
type ClusterResponse struct {
	Cluster struct {
		Nodes     []Node `json:"nodes"`
		Generated string `json:"generated"`
	} `json:"cluster"`
}

// Node ...
type Node struct {
	NodeID              string              `json:"nodeId"`
	Address             string              `json:"address"`
	APIPort             uint                `json:"apiPort"`
	Status              string              `json:"status"`
	Heartbeat           string              `json:"heartbeat"`
	ConnectionRequested string              `json:"connectionRequested"`
	Roles               []string            `json:"roles"`
	ActiveThreadCount   uint64              `json:"activeThreadCount"`
	Queued              string              `json:"queued"`
	Events              []map[string]string `json:"events"`
	NodeStartTime       string              `json:"nodeStartTime"`
}

// GetNodeMap ...
func GetNodeMap(host string, client *http.Client) (map[string]string, error) {

	url := fmt.Sprintf("http://%s/nifi-api/controller/cluster", host)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logp.Err(err.Error())
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		logp.Err(err.Error())
		return nil, err
	}

	defer resp.Body.Close()

	nodeMap := map[string]string{}

	if resp.StatusCode == 200 {
		var data ClusterResponse

		err := json.NewDecoder(resp.Body).Decode(&data)
		if err != nil {
			logp.Err("Error: ", err)
			return nil, err
		}

		for _, node := range data.Cluster.Nodes {
			nodeMap[fmt.Sprintf("%s:%d", node.Address, node.APIPort)] = node.NodeID
		}
		return nodeMap, nil
	}

	return nodeMap, nil
}
