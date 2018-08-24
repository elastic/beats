// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"github.com/elastic/beats/libbeat/cmd/instance"
	"github.com/elastic/beats/x-pack/libbeat/management/api"
)

// Enroll this beat to the given kibana
// This will use Central Management API to enroll and retrieve an access key for config retrieval
func Enroll(beat *instance.Beat, kibanaURL, enrollmentToken string) error {
	config, err := api.ConfigFromURL(kibanaURL)
	if err != nil {
		return err
	}

	client, err := api.NewClient(config)
	if err != nil {
		return err
	}

	accessToken, err := client.Enroll(beat.Info.Beat, beat.Info.Version, beat.Info.Hostname, beat.Info.UUID, enrollmentToken)
	if err != nil {
		return err
	}

	// Enrolled, persist state
	// TODO use beat.Keystore() for access_token
	settings := Config{
		Enabled:     true,
		AccessToken: accessToken,
		Kibana:      config,
	}

	return settings.Save()
}
