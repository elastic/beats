// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package otelmanager

import (
	"bytes"
	"encoding/base64"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
)

// NewAgentClient returns a client for the control protocol, connecting with
// the connection info from Elastic Agent as a beat process does.
// connectionSecret is the base64-encoded connection info, in the format beat
// processes read from stdin.
func NewAgentClient(connectionSecret string) (client.V2, error) {
	connInfo, err := base64.StdEncoding.DecodeString(connectionSecret)
	if err != nil {
		return nil, fmt.Errorf("invalid management.agent.connection_secret: %w", err)
	}

	versionInfo := client.VersionInfo{
		Name:      "beat-v2-client-for-receiver",
		BuildHash: version.Commit(),
		Meta: map[string]string{
			"commit":     version.Commit(),
			"build_time": version.BuildTime().String(),
		},
	}
	agentClient, _, err := client.NewV2FromReader(bytes.NewReader(connInfo), versionInfo)
	if err != nil {
		return nil, fmt.Errorf("error creating Elastic Agent client: %w", err)
	}
	return agentClient, nil
}
