// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

// AgentInfo is a collection of information about agent.
type AgentInfo struct {
	agentID string
}

// NewAgentInfo creates a new agent information.
// Generates new unique identifier for agent.
func NewAgentInfo() (*AgentInfo, error) {
	agentID, err := generateAgentID()
	if err != nil {
		return nil, err
	}

	return &AgentInfo{
		agentID: agentID,
	}, nil
}

// AgentID returns an agent identifier.
func (i *AgentInfo) AgentID() string {
	return i.agentID
}
