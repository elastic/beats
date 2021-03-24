// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/kibana"
)

type localConfig struct {
	Fleet    *configuration.FleetAgentConfig `config:"fleet"`
	Settings *configuration.SettingsConfig   `config:"agent" yaml:"agent"`
}

func createFleetConfigFromEnroll(accessAPIKey string, kbn *kibana.Config) (*configuration.FleetAgentConfig, error) {
	cfg := configuration.DefaultFleetAgentConfig()
	cfg.Enabled = true
	cfg.AccessAPIKey = accessAPIKey
	cfg.Kibana = kbn

	if err := cfg.Valid(); err != nil {
		return nil, errors.New(err, "invalid enrollment options", errors.TypeConfig)
	}
	return cfg, nil
}

func createFleetServerBootstrapConfig(connStr string, policyID string, host string, port uint16, cert string, key string, esCA string) (*configuration.FleetAgentConfig, error) {
	es, err := configuration.ElasticsearchFromConnStr(connStr)
	if err != nil {
		return nil, err
	}
	if esCA != "" {
		es.TLS = &tlscommon.Config{
			CAs: []string{esCA},
		}
	}
	cfg := configuration.DefaultFleetAgentConfig()
	cfg.Enabled = true
	cfg.Server = &configuration.FleetServerConfig{
		Bootstrap: true,
		Output: configuration.FleetServerOutputConfig{
			Elasticsearch: es,
		},
		Host: host,
		Port: port,
	}
	if policyID != "" {
		cfg.Server.Policy = &configuration.FleetServerPolicyConfig{ID: policyID}
	}
	if cert != "" || key != "" {
		cfg.Server.TLS = &tlscommon.Config{
			Certificate: tlscommon.CertificateConfig{
				Certificate: cert,
				Key:         key,
			},
		}
	}

	if err := cfg.Valid(); err != nil {
		return nil, errors.New(err, "invalid enrollment options", errors.TypeConfig)
	}
	return cfg, nil
}
