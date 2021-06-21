// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import "time"

// setup configuration

type setupConfig struct {
	Fleet       fleetConfig       `config:"fleet"`
	FleetServer fleetServerConfig `config:"fleet_server"`
	Kibana      kibanaConfig      `config:"kibana"`
}

type fleetConfig struct {
	CA              string `config:"ca"`
	Enroll          bool   `config:"enroll"`
	EnrollmentToken string `config:"enrollment_token"`
	Force           bool   `config:"force"`
	Insecure        bool   `config:"insecure"`
	TokenName       string `config:"token_name"`
	TokenPolicyName string `config:"token_policy_name"`
	URL             string `config:"url"`
}

type fleetServerConfig struct {
	Cert          string              `config:"cert"`
	CertKey       string              `config:"cert_key"`
	Elasticsearch elasticsearchConfig `config:"elasticsearch"`
	Enable        bool                `config:"enable"`
	Host          string              `config:"host"`
	InsecureHTTP  bool                `config:"insecure_http"`
	PolicyID      string              `config:"policy_id"`
	Port          string              `config:"port"`
	Headers       map[string]string   `config:"headers"`
}

type elasticsearchConfig struct {
	CA           string `config:"ca"`
	Host         string `config:"host"`
	Username     string `config:"username"`
	Password     string `config:"password"`
	ServiceToken string `config:"service_token"`
}

type kibanaConfig struct {
	Fleet              kibanaFleetConfig `config:"fleet"`
	RetrySleepDuration time.Duration     `config:"retry_sleep_duration"`
	RetryMaxCount      int               `config:"retry_max_count"`
	Headers            map[string]string `config:"headers"`
}

type kibanaFleetConfig struct {
	CA       string `config:"ca"`
	Host     string `config:"host"`
	Password string `config:"password"`
	Setup    bool   `config:"setup"`
	Username string `config:"username"`
}

func defaultAccessConfig() (setupConfig, error) {
	retrySleepDuration, err := envDurationWithDefault(defaultRequestRetrySleep, requestRetrySleepEnv)
	if err != nil {
		return setupConfig{}, err
	}

	retryMaxCount, err := envIntWithDefault(defaultMaxRequestRetries, maxRequestRetriesEnv)
	if err != nil {
		return setupConfig{}, err
	}

	cfg := setupConfig{
		Fleet: fleetConfig{
			CA:              envWithDefault("", "FLEET_CA", "KIBANA_CA", "ELASTICSEARCH_CA"),
			Enroll:          envBool("FLEET_ENROLL", "FLEET_SERVER_ENABLE"),
			EnrollmentToken: envWithDefault("", "FLEET_ENROLLMENT_TOKEN"),
			Force:           envBool("FLEET_FORCE"),
			Insecure:        envBool("FLEET_INSECURE"),
			TokenName:       envWithDefault("Default", "FLEET_TOKEN_NAME"),
			TokenPolicyName: envWithDefault("", "FLEET_TOKEN_POLICY_NAME"),
			URL:             envWithDefault("", "FLEET_URL"),
		},
		FleetServer: fleetServerConfig{
			Cert:    envWithDefault("", "FLEET_SERVER_CERT"),
			CertKey: envWithDefault("", "FLEET_SERVER_CERT_KEY"),
			Elasticsearch: elasticsearchConfig{
				Host:         envWithDefault("http://elasticsearch:9200", "FLEET_SERVER_ELASTICSEARCH_HOST", "ELASTICSEARCH_HOST"),
				Username:     envWithDefault("elastic", "FLEET_SERVER_ELASTICSEARCH_USERNAME", "ELASTICSEARCH_USERNAME"),
				Password:     envWithDefault("changeme", "FLEET_SERVER_ELASTICSEARCH_PASSWORD", "ELASTICSEARCH_PASSWORD"),
				ServiceToken: envWithDefault("", "FLEET_SERVER_SERVICE_TOKEN"),
				CA:           envWithDefault("", "FLEET_SERVER_ELASTICSEARCH_CA", "ELASTICSEARCH_CA"),
			},
			Enable:       envBool("FLEET_SERVER_ENABLE"),
			Host:         envWithDefault("", "FLEET_SERVER_HOST"),
			InsecureHTTP: envBool("FLEET_SERVER_INSECURE_HTTP"),
			PolicyID:     envWithDefault("", "FLEET_SERVER_POLICY_ID", "FLEET_SERVER_POLICY"),
			Port:         envWithDefault("", "FLEET_SERVER_PORT"),
			Headers:      envMap("FLEET_HEADER"),
		},
		Kibana: kibanaConfig{
			Fleet: kibanaFleetConfig{
				// Remove FLEET_SETUP in 8.x
				// The FLEET_SETUP environment variable boolean is a fallback to the old name. The name was updated to
				// reflect that its setting up Fleet in Kibana versus setting up Fleet Server.
				Setup:    envBool("KIBANA_FLEET_SETUP", "FLEET_SETUP"),
				Host:     envWithDefault("http://kibana:5601", "KIBANA_FLEET_HOST", "KIBANA_HOST"),
				Username: envWithDefault("elastic", "KIBANA_FLEET_USERNAME", "KIBANA_USERNAME", "ELASTICSEARCH_USERNAME"),
				Password: envWithDefault("changeme", "KIBANA_FLEET_PASSWORD", "KIBANA_PASSWORD", "ELASTICSEARCH_PASSWORD"),
				CA:       envWithDefault("", "KIBANA_FLEET_CA", "KIBANA_CA", "ELASTICSEARCH_CA"),
			},
			RetrySleepDuration: retrySleepDuration,
			RetryMaxCount:      retryMaxCount,
			Headers:            envMap("FLEET_KIBANA_HEADER"),
		},
	}
	return cfg, nil
}
