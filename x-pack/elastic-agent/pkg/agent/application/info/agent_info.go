// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package info

import (
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
)

// AgentInfo is a collection of information about agent.
type AgentInfo struct {
	agentID  string
	logLevel string
}

// NewAgentInfoWithLog creates a new agent information.
// In case when agent ID was already created it returns,
// this created ID otherwise it generates
// new unique identifier for agent.
// If agent config file does not exist it gets created.
// Initiates log level to predefined value.
func NewAgentInfoWithLog(level string) (*AgentInfo, error) {
	agentInfo, err := loadAgentInfo(false, level)
	if err != nil {
		return nil, err
	}

	return &AgentInfo{
		agentID:  agentInfo.ID,
		logLevel: agentInfo.LogLevel,
	}, nil
}

// NewAgentInfo creates a new agent information.
// In case when agent ID was already created it returns,
// this created ID otherwise it generates
// new unique identifier for agent.
// If agent config file does not exist it gets created.
func NewAgentInfo() (*AgentInfo, error) {
	return NewAgentInfoWithLog(defaultLogLevel)
}

// LogLevel updates log level of agent.
func (i *AgentInfo) LogLevel(level string) error {
	if err := updateLogLevel(level); err != nil {
		return err
	}

	i.logLevel = level
	return nil
}

// AgentID returns an agent identifier.
func (i *AgentInfo) AgentID() string {
	return i.agentID
}

// Version returns the version for this Agent.
func (*AgentInfo) Version() string {
	return release.Version()
}

// Snapshot returns if this version is a snapshot.
func (*AgentInfo) Snapshot() bool {
	return release.Snapshot()
}
