// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package info

import (
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
)

// AgentInfo is a collection of information about agent.
type AgentInfo struct {
	agentID  string
	logLevel string
	headers  map[string]string
}

// NewAgentInfoWithLog creates a new agent information.
// In case when agent ID was already created it returns,
// this created ID otherwise it generates
// new unique identifier for agent.
// If agent config file does not exist it gets created.
// Initiates log level to predefined value.
func NewAgentInfoWithLog(level string, createAgentID bool) (*AgentInfo, error) {
	agentInfo, err := loadAgentInfoWithBackoff(false, level, createAgentID)
	if err != nil {
		return nil, err
	}

	return &AgentInfo{
		agentID:  agentInfo.ID,
		logLevel: agentInfo.LogLevel,
		headers:  agentInfo.Headers,
	}, nil
}

// NewAgentInfo creates a new agent information.
// In case when agent ID was already created it returns,
// this created ID otherwise it generates
// new unique identifier for agent.
// If agent config file does not exist it gets created.
func NewAgentInfo(createAgentID bool) (*AgentInfo, error) {
	return NewAgentInfoWithLog(defaultLogLevel, createAgentID)
}

// LogLevel retrieves a log level.
func (i *AgentInfo) LogLevel() string {
	if i.logLevel == "" {
		return logger.DefaultLogLevel.String()
	}
	return i.logLevel
}

// SetLogLevel updates log level of agent.
func (i *AgentInfo) SetLogLevel(level string) error {
	if err := updateLogLevel(level); err != nil {
		return err
	}

	i.logLevel = level
	return nil
}

// ReloadID reloads agent info ID from configuration file.
func (i *AgentInfo) ReloadID() error {
	newInfo, err := NewAgentInfoWithLog(i.logLevel, false)
	if err != nil {
		return err
	}
	i.agentID = newInfo.agentID
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

// Headers returns custom headers used to communicate with elasticsearch.
func (i *AgentInfo) Headers() map[string]string {
	return i.headers
}
