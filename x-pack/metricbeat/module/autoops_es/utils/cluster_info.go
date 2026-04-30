// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package utils

import (
	"github.com/elastic/elastic-agent-libs/version"
)

type ClusterInfoVersion struct {
	Number       *version.V `json:"number"`
	Distribution string     `json:"distribution,omitempty"`
}

// Info construct contains the data from the cluster / endpoint
type ClusterInfo struct {
	ClusterName string             `json:"cluster_name"`
	ClusterID   string             `json:"cluster_uuid"`
	Version     ClusterInfoVersion `json:"version"`
}

// ClusterInfoError represents an error related to cluster information retrieval.
type ClusterInfoError struct {
	Message string
}

// Error implements the error interface for ClusterInfoError.
func (e *ClusterInfoError) Error() string {
	return e.Message
}
