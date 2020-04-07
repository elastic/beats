// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package info

// AgentInfo is a collection of information about agent.
type AgentInfo struct {
	agentID string
}

// NewAgentInfo creates a new agent information.
// In case when agent ID was already created it returns,
// this created ID otherwise it generates
// new unique identifier for agent.
// If agent config file does not exist it gets created.
func NewAgentInfo() (*AgentInfo, error) {
	agentInfo, err := loadAgentInfo(false)
	if err != nil {
		return nil, err
	}

	return &AgentInfo{
		agentID: agentInfo.ID,
	}, nil
}

// ForceNewAgentInfo creates a new agent information.
// Generates new unique identifier for agent regardless
// of any existing ID.
// If agent config file does not exist it gets created.
func ForceNewAgentInfo() (*AgentInfo, error) {
	agentInfo, err := loadAgentInfo(true)
	if err != nil {
		return nil, err
	}

	return &AgentInfo{
		agentID: agentInfo.ID,
	}, nil
}

// AgentID returns an agent identifier.
func (i *AgentInfo) AgentID() string {
	return i.agentID
}
