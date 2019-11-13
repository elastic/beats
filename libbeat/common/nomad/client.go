package nomad

import (
	"fmt"
	"net/http"

	nomad "github.com/hashicorp/nomad/api"
)

type NomadClient interface {
	Allocations(nodeID string, q *nomad.QueryOptions) ([]*nomad.Allocation, *nomad.QueryMeta, error)
}

type apiClient struct {
	client *nomad.Client
}

func (c *apiClient) Allocations(nodeID string, q *nomad.QueryOptions) ([]*nomad.Allocation, *nomad.QueryMeta, error) {
	return c.client.Nodes().Allocations(nodeID, q)
}

func WrapClient(client *nomad.Client) NomadClient {
	return &apiClient{client: client}
}

func NewClient(address, region, secretID string, httpClient *http.Client) (*nomad.Client, error) {
	cfg := nomad.Config{
		Address:    address,
		Region:     region,
		SecretID:   secretID,
		HttpClient: httpClient,
	}
	return nomad.NewClient(&cfg)
}

// GetLocalNodeID returns the node ID of the local Nomad Client and an error if
// it couldn't be determined or the Agent is not running in Client mode.
func GetLocalNodeID(client *nomad.Client) (string, error) {
	info, err := client.Agent().Self()
	if err != nil {
		return "", fmt.Errorf("Error querying agent info: %s", err)
	}
	clientStats, ok := info.Stats["client"]
	if !ok {
		return "", fmt.Errorf("Nomad not running in client mode")
	}

	nodeID, ok := clientStats["node_id"]
	if !ok {
		return "", fmt.Errorf("Failed to determine node ID")
	}

	return nodeID, nil
}