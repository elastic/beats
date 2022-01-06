// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package plugin

import (
	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

type configFetcher interface {
	Config() string
}

// IsRestartNeeded returns true if
// - spec is configured to support restart on change
// - output changes in between configs
func IsRestartNeeded(log *logger.Logger, spec program.Spec, cfgFetch configFetcher, newCfg map[string]interface{}) bool {
	if !spec.RestartOnOutputChange {
		// early exit if restart is not needed anyway
		return false
	}

	// compare outputs
	curCfgStr := cfgFetch.Config()
	if curCfgStr == "" {
		// no config currently applied
		return false
	}

	currentOutput, err := getOutputConfigFromString(curCfgStr)
	if err != nil {
		log.Errorf("failed to retrieve output config from current state: %v", err)
		return false
	}

	newOutput, err := getOutputConfigFromMap(newCfg)
	if err != nil {
		log.Errorf("failed to retrieve output config from new state: %v", err)
		return false
	}

	// restart needed only if output changed
	return currentOutput != newOutput
}

func getOutputConfigFromString(cfgString string) (string, error) {
	cfg, err := config.NewConfigFrom(cfgString)
	if err != nil {
		return "", err
	}

	cfgMap, err := cfg.ToMapStr()
	if err != nil {
		return "", err
	}

	return getOutputConfigFromMap(cfgMap)
}

func getOutputConfigFromMap(cfgMap map[string]interface{}) (string, error) {
	outputCfgIface, found := cfgMap["output"]
	if !found {
		// output not found not an error
		return "", nil
	}

	outputCfg, ok := outputCfgIface.(map[string]interface{})
	if !ok {
		return "", errors.New("not a map")
	}

	cfgStr, err := yaml.Marshal(outputCfg)
	if err != nil {
		return "", errors.New(err, errors.TypeApplication)
	}

	return string(cfgStr), nil
}
