// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"fmt"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/feature"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/unison"
)

const (
	expandEventListFromField = "records"
	inputName                = "azure-eventhub"
	processorV1              = "v1"
	processorV2              = "v2"
)

// Plugin returns the Azure Event Hub input plugin.
//
// Required register the plugin loader for the
// input API v2.
func Plugin(log *logp.Logger) v2.Plugin {
	return v2.Plugin{
		Name:       inputName,
		Stability:  feature.Stable,
		Deprecated: false,
		Info:       "Collect logs from Azure Event Hub",
		Manager: &eventHubInputManager{
			log: log,
		},

		// ExcludeFromFIPS = true to prevent this input from being used in FIPS-capable
		// Filebeat distributions. This input indirectly uses algorithms that are not
		// FIPS-compliant. Specifically, the input depends on the
		// github.com/Azure/azure-sdk-for-go/sdk/azidentity package which, in turn,
		// depends on the golang.org/x/crypto/pkcs12 package, which is not FIPS-compliant.
		//
		// TODO: investigate whether FIPS exclusion is still needed now that
		// the deprecated azure-event-hubs-go SDK has been removed.
		ExcludeFromFIPS: true,
	}
}

// azureInputConfig is responsible for creating the right azure-eventhub input
// based on the configuration.
type eventHubInputManager struct {
	log *logp.Logger
}

// Init initializes the input manager.
func (m *eventHubInputManager) Init(unison.Group) error {
	return nil
}

// Create creates a new azure-eventhub input based on the configuration.
func (m *eventHubInputManager) Create(cfg *conf.C) (v2.Input, error) {

	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, fmt.Errorf("reading %s input config: %w", inputName, err)
	}

	config.checkUnsupportedParams(m.log)

	return newEventHubInputV2(config, m.log)
}
