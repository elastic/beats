// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package info

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/gofrs/uuid"
	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/filelock"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/storage"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/backoff"
	monitoringConfig "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/monitoring/config"
)

// defaultAgentConfigFile is a name of file used to store agent information
const agentInfoKey = "agent"
const defaultLogLevel = "info"
const maxRetriesloadAgentInfo = 5

type persistentAgentInfo struct {
	ID             string                                 `json:"id" yaml:"id" config:"id"`
	Headers        map[string]string                      `json:"headers" yaml:"headers" config:"headers"`
	LogLevel       string                                 `json:"logging.level,omitempty" yaml:"logging.level,omitempty" config:"logging.level,omitempty"`
	MonitoringHTTP *monitoringConfig.MonitoringHTTPConfig `json:"monitoring.http,omitempty" yaml:"monitoring.http,omitempty" config:"monitoring.http,omitempty"`
}

type ioStore interface {
	Save(io.Reader) error
	Load() (io.ReadCloser, error)
}

// updateLogLevel updates log level and persists it to disk.
func updateLogLevel(level string) error {
	ai, err := loadAgentInfoWithBackoff(false, defaultLogLevel, false)
	if err != nil {
		return err
	}

	if ai.LogLevel == level {
		// no action needed
		return nil
	}

	agentConfigFile := paths.AgentConfigFile()
	diskStore := storage.NewDiskStore(agentConfigFile)

	ai.LogLevel = level
	return updateAgentInfo(diskStore, ai)
}

func generateAgentID() (string, error) {
	uid, err := uuid.NewV4()
	if err != nil {
		return "", fmt.Errorf("error while generating UUID for agent: %v", err)
	}

	return uid.String(), nil
}

func getInfoFromStore(s ioStore, logLevel string) (*persistentAgentInfo, error) {
	agentConfigFile := paths.AgentConfigFile()
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
		return &persistentAgentInfo{
			LogLevel:       logLevel,
			MonitoringHTTP: monitoringConfig.DefaultConfig().HTTP,
		}, nil
	}

	cc, err := config.NewConfigFrom(agentInfoSubMap)
	if err != nil {
		return nil, errors.New(err, "failed to create config from agent info submap")
	}

	pid := &persistentAgentInfo{
		LogLevel:       logLevel,
		MonitoringHTTP: monitoringConfig.DefaultConfig().HTTP,
	}
	if err := cc.Unpack(&pid); err != nil {
		return nil, errors.New(err, "failed to unpack stored config to map")
	}

	return pid, nil
}

func updateAgentInfo(s ioStore, agentInfo *persistentAgentInfo) error {
	agentConfigFile := paths.AgentConfigFile()
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

	// best effort to keep the ID
	if agentInfoSubMap, found := configMap[agentInfoKey]; found {
		if cc, err := config.NewConfigFrom(agentInfoSubMap); err == nil {
			pid := &persistentAgentInfo{}
			err := cc.Unpack(&pid)
			if err == nil && pid.ID != agentInfo.ID {
				// if our id is different (we just generated it)
				// keep the one present in the file
				agentInfo.ID = pid.ID
			}
		}
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

func loadAgentInfoWithBackoff(forceUpdate bool, logLevel string, createAgentID bool) (*persistentAgentInfo, error) {
	var err error
	var ai *persistentAgentInfo

	signal := make(chan struct{})
	backExp := backoff.NewExpBackoff(signal, 100*time.Millisecond, 3*time.Second)

	for i := 0; i <= maxRetriesloadAgentInfo; i++ {
		backExp.Wait()
		ai, err = loadAgentInfo(forceUpdate, logLevel, createAgentID)
		if err != filelock.ErrAppAlreadyRunning {
			break
		}
	}

	close(signal)
	return ai, err
}

func loadAgentInfo(forceUpdate bool, logLevel string, createAgentID bool) (*persistentAgentInfo, error) {
	idLock := paths.AgentConfigFileLock()
	if err := idLock.TryLock(); err != nil {
		return nil, err
	}
	defer idLock.Unlock()

	agentConfigFile := paths.AgentConfigFile()
	diskStore := storage.NewDiskStore(agentConfigFile)

	agentinfo, err := getInfoFromStore(diskStore, logLevel)
	if err != nil {
		return nil, err
	}

	if agentinfo != nil && !forceUpdate && (agentinfo.ID != "" || !createAgentID) {
		return agentinfo, nil
	}

	if err := updateID(agentinfo, diskStore); err != nil {
		return nil, err
	}

	return agentinfo, nil
}

func updateID(agentInfo *persistentAgentInfo, s ioStore) error {
	var err error
	agentInfo.ID, err = generateAgentID()
	if err != nil {
		return err
	}

	if err := updateAgentInfo(s, agentInfo); err != nil {
		return errors.New(err, "storing generated agent id", errors.TypeFilesystem)
	}

	return nil
}
