// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"os"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/cmd/instance"
	"github.com/elastic/beats/x-pack/libbeat/management/api"
)

const accessTokenK = "management.accesstoken"

// Enroll this beat to the given kibana
// This will use Central Management API to enroll and retrieve an access key for config retrieval
func Enroll(beat *instance.Beat, kibanaURL, enrollmentToken string) error {
	kibanaConfig, err := api.ConfigFromURL(kibanaURL)
	if err != nil {
		return err
	}

	// Ignore kibana version to avoid permission errors
	kibanaConfig.IgnoreVersion = true

	client, err := api.NewClient(kibanaConfig)
	if err != nil {
		return err
	}

	accessToken, err := client.Enroll(beat.Info.Beat, beat.Info.Name, beat.Info.Version, beat.Info.Hostname, beat.Info.UUID, enrollmentToken)
	if err != nil {
		return err
	}

	// Store access token in keystore
	if err := storeAccessToken(beat, accessToken); err != nil {
		return err
	}

	// Enrolled, persist state
	config := defaultConfig()
	config.Enabled = true
	config.AccessToken = "${" + accessTokenK + "}"
	config.Kibana = kibanaConfig

	// TODO ask for confirmation before doing this, save a backup copy:
	configFile := cfgfile.GetDefaultCfgfile()
	f, err := os.OpenFile(configFile, os.O_RDWR|os.O_TRUNC, 0600)
	if err != nil {
		return errors.Wrap(err, "opening settings file")
	}
	defer f.Close()

	if err := config.OverwriteConfigFile(f, beat.Beat.Info.Beat); err != nil {
		return errors.Wrap(err, "overriding settings file")
	}

	return nil
}

func storeAccessToken(beat *instance.Beat, accessToken string) error {
	keystore := beat.Keystore()
	if !keystore.IsPersisted() {
		if err := keystore.Create(false); err != nil {
			return errors.Wrap(err, "error creating keystore")
		}
	}

	if err := keystore.Store(accessTokenK, []byte(accessToken)); err != nil {
		return errors.Wrap(err, "error storing the access token")
	}

	return keystore.Save()
}
