// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/cmd/instance"
	"github.com/elastic/beats/libbeat/common/file"
	"github.com/elastic/beats/libbeat/kibana"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/management"
	"github.com/elastic/beats/x-pack/libbeat/management/api"
)

const accessTokenKey = "management.accesstoken"

// Enroll this beat to the given kibana
// This will use Central Management API to enroll and retrieve an access key for config retrieval
func Enroll(
	beat *instance.Beat,
	kibanaConfig *kibana.ClientConfig,
	enrollmentToken string,
) error {
	// Ignore kibana version to avoid permission errors
	kibanaConfig.IgnoreVersion = true

	client, err := api.NewClient(kibanaConfig)
	if err != nil {
		return err
	}

	logp.NewLogger(management.DebugK).Warn("DEPRECATED: Central Management is no longer under development and has been deprecated. We are working hard to deliver a new and more comprehensive solution and look forward to sharing it with you")

	accessToken, err := client.Enroll(beat.Info.Beat, beat.Info.Name, beat.Info.Version, beat.Info.Hostname, beat.Info.ID, enrollmentToken)
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
	config.AccessToken = "${" + accessTokenKey + "}"
	config.Kibana = kibanaConfig

	configFile := cfgfile.GetDefaultCfgfile()

	ts := time.Now()

	// This timestamp format is a variation of RFC3339 replacing colons with
	// slashes so that it can be used as part of a filename in all OSes.
	// (Colon is not a valid character for filenames in Windows).
	// Also removed the TZ-offset as that can cause a plus sign to be output.
	const fsSafeTimestamp = "2006-01-02T15-04-05"

	// backup current settings:
	backConfigFile := configFile + "." + ts.Format(fsSafeTimestamp) + ".bak"
	fmt.Println("Saving a copy of current settings to " + backConfigFile)
	err = file.SafeFileRotate(backConfigFile, configFile)
	if err != nil {
		return errors.Wrap(err, "creating a backup copy of current settings")
	}

	// create the new ones:
	f, err := os.OpenFile(configFile, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0600)
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

	if err := keystore.Store(accessTokenKey, []byte(accessToken)); err != nil {
		return errors.Wrap(err, "error storing the access token")
	}

	return keystore.Save()
}
