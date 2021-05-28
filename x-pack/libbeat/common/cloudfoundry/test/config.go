// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package test

import (
	"os"
	"testing"

	"github.com/gofrs/uuid"
)

func GetConfigFromEnv(t *testing.T) map[string]interface{} {
	t.Helper()

	shardID, err := uuid.NewV4()
	if err != nil {
		t.Fatalf("Unable to create a random shard ID: %v", err)
	}

	config := map[string]interface{}{
		"api_address":   lookupEnv(t, "CLOUDFOUNDRY_API_ADDRESS"),
		"client_id":     lookupEnv(t, "CLOUDFOUNDRY_CLIENT_ID"),
		"client_secret": lookupEnv(t, "CLOUDFOUNDRY_CLIENT_SECRET"),
		"shard_id":      shardID.String(),

		"ssl.verification_mode": "none",
	}

	optionalConfig(config, "uaa_address", "CLOUDFOUNDRY_UAA_ADDRESS")
	optionalConfig(config, "rlp_address", "CLOUDFOUNDRY_RLP_ADDRESS")
	optionalConfig(config, "doppler_address", "CLOUDFOUNDRY_DOPPLER_ADDRESS")

	if t.Failed() {
		t.FailNow()
	}

	return config
}

func lookupEnv(t *testing.T, name string) string {
	value, ok := os.LookupEnv(name)
	if !ok {
		t.Errorf("Environment variable %s is not set", name)
	}
	return value
}

func optionalConfig(config map[string]interface{}, key string, envVar string) {
	if value, ok := os.LookupEnv(envVar); ok {
		config[key] = value
	}
}
