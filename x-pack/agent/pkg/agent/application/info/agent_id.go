// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package info

import (
	"bytes"
	"fmt"
	"io"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/storage"
	"github.com/elastic/beats/x-pack/agent/pkg/config"
)

// AgentConfigFile is a name of file used to store agent information
const AgentConfigFile = "fleet.yml"
const agentInfoKey = "agent_info"

type persistentAgentInfo struct {
	ID string `json:"ID" yaml:"ID" config:"ID"`
}

type ioStore interface {
	Save(io.Reader) error
	Load() (io.ReadCloser, error)
}

func generateAgentID() (string, error) {
	uid, err := uuid.NewV4()
	if err != nil {
		return "", fmt.Errorf("error while generating UUID for agent: %v", err)
	}

	return uid.String(), nil
}

func loadAgentInfo(forceUpdate bool) (*persistentAgentInfo, error) {
	s := storage.NewEncryptedDiskStore(AgentConfigFile, []byte(""))

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
		return nil, errors.Wrap(err, "storing generated agent id")
	}

	return agentinfo, nil
}

func getInfoFromStore(s ioStore) (*persistentAgentInfo, error) {
	reader, err := s.Load()
	if err != nil {
		return nil, err
	}

	cfg, err := config.NewConfigFrom(reader)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to read configuration %s for the agent", AgentConfigFile)
	}

	configMap, err := cfg.ToMapStr()
	if err != nil {
		return nil, errors.Wrap(err, "failed to unpack stored config to map")
	}

	agentInfoSubMap, found := configMap[agentInfoKey]
	if !found {
		return &persistentAgentInfo{}, nil
	}

	cc, err := config.NewConfigFrom(agentInfoSubMap)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create config from agent info submap")
	}

	pid := &persistentAgentInfo{}
	if err := cc.Unpack(&pid); err != nil {
		return nil, errors.Wrap(err, "failed to unpack stored config to map")
	}

	return pid, nil
}

func updateAgentInfo(s ioStore, agentInfo *persistentAgentInfo) error {
	reader, err := s.Load()
	if err != nil {
		return err
	}

	cfg, err := config.NewConfigFrom(reader)
	if err != nil {
		return errors.Wrapf(err, "fail to read configuration %s for the agent", AgentConfigFile)
	}

	configMap := make(map[string]interface{})
	if err := cfg.Unpack(&configMap); err != nil {
		return errors.Wrap(err, "failed to unpack stored config to map")
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
		return nil, errors.Wrap(err, "could not marshal to YAML")
	}
	return bytes.NewReader(data), nil
}
