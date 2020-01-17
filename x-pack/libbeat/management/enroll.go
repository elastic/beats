// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"fmt"
	"os"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/cmd/instance"
	"github.com/elastic/beats/libbeat/common/cli"
	"github.com/elastic/beats/libbeat/common/file"
	"github.com/elastic/beats/libbeat/kibana"
	"github.com/elastic/beats/x-pack/libbeat/management/api"
)

const accessTokenKey = "management.accesstoken"

// Enroll this beat to the given kibana
// This will use Central Management API to enroll and retrieve an access key for config retrieval
func Enroll(beat *instance.Beat, kibanaConfig *kibana.ClientConfig, enrollmentToken string, force bool) (bool, error) {
	// Ignore kibana version to avoid permission errors
	kibanaConfig.IgnoreVersion = true

	client, err := api.NewClient(kibanaConfig)
	if err != nil {
		return false, err
	}

	accessToken, err := client.Enroll(beat.Info.Beat, beat.Info.Name, beat.Info.Version, beat.Info.Hostname, beat.Info.UUID, enrollmentToken)
	if err != nil {
		return false, err
	}

	// Store access token in keystore
	if err := storeAccessToken(beat, accessToken); err != nil {
		return false, err
	}

	// Enrolled, persist state
	config := defaultConfig()
	config.Enabled = true
	config.AccessToken = "${" + accessTokenKey + "}"
	config.Kibana = kibanaConfig

	confirm, err := confirmConfigOverwrite(force)
	if err != nil {
		return false, err
	}

	if confirm {
		configFile := cfgfile.GetDefaultCfgfile()

		// backup current settings:
		backConfigFile := configFile + ".bak"
		fmt.Println("Saving a copy of current settings to " + backConfigFile)
		err := file.SafeFileRotate(backConfigFile, configFile)
		if err != nil {
			return false, errors.Wrap(err, "creating a backup copy of current settings")
		}

		// create the new ones:
		f, err := os.OpenFile(configFile, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0600)
		if err != nil {
			return false, errors.Wrap(err, "opening settings file")
		}
		defer f.Close()

		if err := config.OverwriteConfigFile(f, beat.Beat.Info.Beat); err != nil {
			return false, errors.Wrap(err, "overriding settings file")
		}
	}

	return true, nil
}

func storeAccessToken(beat *instance.Beat, accessToken string) error {
	keystore := beat.Keystore()
	if !keystore.IsPersisted() {
		if err := keystore.Create(false); err != nil {
			return errors.Wrap(err, "error creating keystore")
		}
	}

	if err := keystore.Store(accessTokenKey, []byte(accessToken)); err != nil {
		return errors.Wrap(err, "error storing the access token")
	}

	return keystore.Save()
}

func confirmConfigOverwrite(force bool) (bool, error) {
	if force {
		return true, nil
	}

	return cli.Confirm("This will replace your current settings. Do you want to continue?", true)
}
