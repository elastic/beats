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
	CA              string        `config:"ca"`
	Enroll          bool          `config:"enroll"`
	EnrollmentToken string        `config:"enrollment_token"`
	Force           bool          `config:"force"`
	Insecure        bool          `config:"insecure"`
	TokenName       string        `config:"token_name"`
	TokenPolicyName string        `config:"token_policy_name"`
	URL             string        `config:"url"`
	DaemonTimeout   time.Duration `config:"daemon_timeout"`
}

type fleetServerConfig struct {
	Cert            string              `config:"cert"`
	CertKey         string              `config:"cert_key"`
	Elasticsearch   elasticsearchConfig `config:"elasticsearch"`
	Enable          bool                `config:"enable"`
	Host            string              `config:"host"`
	InsecureHTTP    bool                `config:"insecure_http"`
	PolicyID        string              `config:"policy_id"`
	DefaultPolicyID string              `config:"default_policy_id"`
	Port            string              `config:"port"`
	Headers         map[string]string   `config:"headers"`
	Timeout         time.Duration       `config:"timeout"`
}

type elasticsearchConfig struct {
	CA                   string `config:"ca"`
	CATrustedFingerprint string `config:"ca_trusted_fingerprint"`
	Host                 string `config:"host"`
	ServiceToken         string `config:"service_token"`
	Insecure             bool   `config:"insecure"`
}

type kibanaConfig struct {
	Fleet              kibanaFleetConfig `config:"fleet"`
	RetrySleepDuration time.Duration     `config:"retry_sleep_duration"`
	RetryMaxCount      int               `config:"retry_max_count"`
	Headers            map[string]string `config:"headers"`
}

type kibanaFleetConfig struct {
	CA           string `config:"ca"`
	Host         string `config:"host"`
	Setup        bool   `config:"setup"`
	Username     string `config:"username"`
	Password     string `config:"password"`
	ServiceToken string `config:"service_token"`
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
			DaemonTimeout:   envTimeout("FLEET_DAEMON_TIMEOUT"),
		},
		FleetServer: fleetServerConfig{
			Cert:    envWithDefault("", "FLEET_SERVER_CERT"),
			CertKey: envWithDefault("", "FLEET_SERVER_CERT_KEY"),
			Elasticsearch: elasticsearchConfig{
				Host:                 envWithDefault("http://elasticsearch:9200", "FLEET_SERVER_ELASTICSEARCH_HOST", "ELASTICSEARCH_HOST"),
				ServiceToken:         envWithDefault("", "FLEET_SERVER_SERVICE_TOKEN"),
				CA:                   envWithDefault("", "FLEET_SERVER_ELASTICSEARCH_CA", "ELASTICSEARCH_CA"),
				CATrustedFingerprint: envWithDefault("", "FLEET_SERVER_ELASTICSEARCH_CA_TRUSTED_FINGERPRINT"),
				Insecure:             envBool("FLEET_SERVER_ELASTICSEARCH_INSECURE"),
			},
			Enable:          envBool("FLEET_SERVER_ENABLE"),
			Host:            envWithDefault("", "FLEET_SERVER_HOST"),
			InsecureHTTP:    envBool("FLEET_SERVER_INSECURE_HTTP"),
			PolicyID:        envWithDefault("", "FLEET_SERVER_POLICY_ID", "FLEET_SERVER_POLICY"),
			Port:            envWithDefault("", "FLEET_SERVER_PORT"),
			Headers:         envMap("FLEET_HEADER"),
			Timeout:         envTimeout("FLEET_SERVER_TIMEOUT"),
			DefaultPolicyID: envWithDefault("fleet-server-policy", "DEFAULT_FLEET_SERVER_POLICY_ID", "DEFAULT_FLEET_SERVER_POLICY"),
		},
		Kibana: kibanaConfig{
			Fleet: kibanaFleetConfig{
				Setup:        envBool("KIBANA_FLEET_SETUP"),
				Host:         envWithDefault("http://kibana:5601", "KIBANA_FLEET_HOST", "KIBANA_HOST"),
				Username:     envWithDefault("elastic", "KIBANA_FLEET_USERNAME", "KIBANA_USERNAME", "ELASTICSEARCH_USERNAME"),
				Password:     envWithDefault("changeme", "KIBANA_FLEET_PASSWORD", "KIBANA_PASSWORD", "ELASTICSEARCH_PASSWORD"),
				ServiceToken: envWithDefault("", "KIBANA_FLEET_SERVICE_TOKEN", "FLEET_SERVER_SERVICE_TOKEN"),
				CA:           envWithDefault("", "KIBANA_FLEET_CA", "KIBANA_CA", "ELASTICSEARCH_CA"),
			},
			RetrySleepDuration: retrySleepDuration,
			RetryMaxCount:      retryMaxCount,
			Headers:            envMap("FLEET_KIBANA_HEADER"),
		},
	}
	return cfg, nil
}
