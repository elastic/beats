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

package nomad

import (
	"fmt"
	"net/http"

	nomad "github.com/hashicorp/nomad/api"
)

// NomadClient defines the interface that nomad clients have to implement
type NomadClient interface {
	Allocations(nodeID string, q *nomad.QueryOptions) ([]*nomad.Allocation, *nomad.QueryMeta, error)
}

type apiClient struct {
	client *nomad.Client
}

// Allocations returns the allocations present on the node with nodeID
func (c *apiClient) Allocations(nodeID string, q *nomad.QueryOptions) ([]*nomad.Allocation, *nomad.QueryMeta, error) {
	return c.client.Nodes().Allocations(nodeID, q)
}

// WrapClient returns an abstract NomadClient based on the nomad.Client provided
func WrapClient(client *nomad.Client) NomadClient {
	return &apiClient{client: client}
}

// NewClient returns a nomad client with the provided configuration
func NewClient(address, region, secretID string, httpClient *http.Client) (*nomad.Client, error) {
	cfg := nomad.DefaultConfig()
	if address != "" {
		cfg.Address = address
	}

	if region != "" {
		cfg.Region = region
	}

	if secretID != "" {
		cfg.SecretID = secretID
	}

	if httpClient != nil {
		cfg.HttpClient = httpClient
	}
	return nomad.NewClient(cfg)
}

// GetLocalNodeID returns the node ID of the local Nomad Client and an error if
// it couldn't be determined or the Agent is not running in Client mode.
func GetLocalNodeID(client *nomad.Client) (string, error) {
	info, err := client.Agent().Self()
	if err != nil {
		return "", fmt.Errorf("error querying agent info: %s", err)
	}
	clientStats, ok := info.Stats["client"]
	if !ok {
		return "", fmt.Errorf("error getting client info: omad not running in client mode")
	}

	nodeID, ok := clientStats["node_id"]
	if !ok {
		return "", fmt.Errorf("failed to determine node ID")
	}

	return nodeID, nil
}
