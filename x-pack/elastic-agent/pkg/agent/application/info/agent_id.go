// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package info

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"

	"github.com/gofrs/uuid"
	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/storage"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
)

// defaultAgentConfigFile is a name of file used to store agent information
const defaultAgentConfigFile = "fleet.yml"
const agentInfoKey = "agent"

// defaultAgentActionStoreFile is the file that will contains the action that can be replayed after restart.
const defaultAgentActionStoreFile = "action_store.yml"

type persistentAgentInfo struct {
	ID string `json:"id" yaml:"id" config:"id"`
}

type ioStore interface {
	Save(io.Reader) error
	Load() (io.ReadCloser, error)
}

// AgentConfigFile is a name of file used to store agent information
func AgentConfigFile() string {
	return filepath.Join(paths.Config(), defaultAgentConfigFile)
}

// AgentActionStoreFile is the file that will contains the action that can be replayed after restart.
func AgentActionStoreFile() string {
	return filepath.Join(paths.Home(), defaultAgentActionStoreFile)
}

func generateAgentID() (string, error) {
	uid, err := uuid.NewV4()
	if err != nil {
		return "", fmt.Errorf("error while generating UUID for agent: %v", err)
	}

	return uid.String(), nil
}

func loadAgentInfo(forceUpdate bool) (*persistentAgentInfo, error) {
	agentConfigFile := AgentConfigFile()
	s := storage.NewDiskStore(agentConfigFile)

	agentinfo, err := getInfoFromStore(s)
	if err != nil {
		return nil, err
	}

	if agentinfo != nil && !forceUpdate && agentinfo.ID != "" {
		return agentinfo, nil
	}

	agentinfo.ID, err = generateAgentID()
	if err != nil {
		return nil, err
	}

	if err := updateAgentInfo(s, agentinfo); err != nil {
		return nil, errors.New(err, "storing generated agent id", errors.TypeFilesystem)
	}

	return agentinfo, nil
}

func getInfoFromStore(s ioStore) (*persistentAgentInfo, error) {
	agentConfigFile := AgentConfigFile()
	reader, err := s.Load()
	if err != nil {
		return nil, err
	}

	// reader is closed by this function
	cfg, err := config.NewConfigFrom(reader)
	if err != nil {
		return nil, errors.New(err,
			fmt.Sprintf("fail to read configuration %s for the agent", agentConfigFile),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, agentConfigFile))
	}

	configMap, err := cfg.ToMapStr()
	if err != nil {
		return nil, errors.New(err,
			"failed to unpack stored config to map",
			errors.TypeFilesystem)
	}

	agentInfoSubMap, found := configMap[agentInfoKey]
	if !found {
		return &persistentAgentInfo{}, nil
	}

	cc, err := config.NewConfigFrom(agentInfoSubMap)
	if err != nil {
		return nil, errors.New(err, "failed to create config from agent info submap")
	}

	pid := &persistentAgentInfo{}
	if err := cc.Unpack(&pid); err != nil {
		return nil, errors.New(err, "failed to unpack stored config to map")
	}

	return pid, nil
}

func updateAgentInfo(s ioStore, agentInfo *persistentAgentInfo) error {
	agentConfigFile := AgentConfigFile()
	reader, err := s.Load()
	if err != nil {
		return err
	}

	// reader is closed by this function
	cfg, err := config.NewConfigFrom(reader)
	if err != nil {
		return errors.New(err, fmt.Sprintf("fail to read configuration %s for the agent", agentConfigFile),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, agentConfigFile))
	}

	configMap := make(map[string]interface{})
	if err := cfg.Unpack(&configMap); err != nil {
		return errors.New(err, "failed to unpack stored config to map")
	}

	configMap[agentInfoKey] = agentInfo

	r, err := yamlToReader(configMap)
	if err != nil {
		return err
	}

	return s.Save(r)
}

func yamlToReader(in interface{}) (io.Reader, error) {
	data, err := yaml.Marshal(in)
	if err != nil {
		return nil, errors.New(err, "could not marshal to YAML")
	}
	return bytes.NewReader(data), nil
}
