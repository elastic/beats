// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package info

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/storage"
)

func TestLoadAgentInfo(t *testing.T) {
	defer cleanupFile()

	agentInfo, err := loadAgentInfo(false)
	require.NoError(t, err)
	assert.NotEqual(t, "", agentInfo.ID)
	assert.Equal(t, "", agentInfo.CapID)
}

func TestLoadAgentInfoForce(t *testing.T) {
	defer cleanupFile()

	agentInfo, err := loadAgentInfo(false)
	require.NoError(t, err)
	assert.NotEqual(t, "", agentInfo.ID)
	assert.Equal(t, "", agentInfo.CapID)

	agentInfo2, err := loadAgentInfo(true)
	require.NoError(t, err)
	assert.NotEqual(t, agentInfo.ID, agentInfo2.ID)
}

func TestLoadAgentInfoUpgrade(t *testing.T) {
	defer cleanupFile()

	// write an old agent info
	agentConfigFile := AgentConfigFile()
	s := storage.NewEncryptedDiskStore(agentConfigFile, []byte(""))
	id, err := generateAgentID()
	require.NoError(t, err)
	err = writeOldAgentInfo(s, id)
	require.NoError(t, err)

	// load agent info, will handle upgrade
	agentInfo, err := loadAgentInfo(false)
	require.NoError(t, err)
	assert.Equal(t, id, agentInfo.ID)
	assert.Equal(t, "", agentInfo.CapID)
}

func cleanupFile() {
	os.Remove(AgentConfigFile())
}

func writeOldAgentInfo(s ioStore, id string) error {
	configMap := make(map[string]interface{})
	configMap[agentInfoKey] = struct {
		ID string `json:"ID" yaml:"ID" config:"ID"`
	}{
		ID: id,
	}

	r, err := yamlToReader(configMap)
	if err != nil {
		return err
	}

	return s.Save(r)
}
