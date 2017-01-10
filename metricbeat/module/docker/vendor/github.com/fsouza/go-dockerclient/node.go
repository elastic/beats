// Copyright 2016 go-dockerclient authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package docker

import (
	"encoding/json"
	"net/http"

	"github.com/fsouza/go-dockerclient/external/github.com/docker/api/types/swarm"
)

// NoSuchNode is the error returned when a given node does not exist.
type NoSuchNode struct {
	ID  string
	Err error
}

func (err *NoSuchNode) Error() string {
	if err.Err != nil {
		return err.Err.Error()
	}
	return "No such node: " + err.ID
}

// ListNodesOptions specify parameters to the ListNodes function.
//
// See http://goo.gl/3K4GwU for more details.
type ListNodesOptions struct {
	Filters map[string][]string
}

// ListNodes returns a slice of nodes matching the given criteria.
//
// See http://goo.gl/3K4GwU for more details.
func (c *Client) ListNodes(opts ListNodesOptions) ([]swarm.Node, error) {
	path := "/nodes?" + queryString(opts)
	resp, err := c.do("GET", path, doOptions{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var nodes []swarm.Node
	if err := json.NewDecoder(resp.Body).Decode(&nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}

// InspectNode returns information about a node by its ID.
//
// See http://goo.gl/WjkTOk for more details.
func (c *Client) InspectNode(id string) (*swarm.Node, error) {
	resp, err := c.do("GET", "/nodes/"+id, doOptions{})
	if err != nil {
		if e, ok := err.(*Error); ok && e.Status == http.StatusNotFound {
			return nil, &NoSuchNode{ID: id}
		}
		return nil, err
	}
	defer resp.Body.Close()
	var node swarm.Node
	if err := json.NewDecoder(resp.Body).Decode(&node); err != nil {
		return nil, err
	}
	return &node, nil
}
