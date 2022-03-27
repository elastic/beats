// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cometd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Validate that it finds the application default credentials and does
// not trigger a config validation error because credentials were not
// set in the config.
func TestConfigValidate(t *testing.T) {
	c := defaultConfig()
	assert.NoError(t, c.Validate())
}

func TestConfigValidateFailure(t *testing.T) {
	var c config
	c.ChannelName = ""
	assert.Error(t, c.Validate())
}

func TestConfigAuthValidate(t *testing.T) {
	var o oAuth2Config
	o.ClientID = "DEMOCLIENTID"
	o.ClientSecret = "DEMOCLIENTSECRET"
	o.User = "salesforce_user"
	o.Password = "P@$$w0₹D"
	o.TokenURL = "https://login.salesforce.com/services/oauth2/token"

	assert.NoError(t, o.Validate())
}

func TestConfigAuthValidateFailure_MissingTokenURL(t *testing.T) {
	var o oAuth2Config
	o.ClientID = "DEMOCLIENTID"
	o.ClientSecret = "DEMOCLIENTSECRET"
	o.User = "salesforce_user"
	o.Password = "P@$$w0₹D"

	assert.Error(t, o.Validate())
}

func TestConfigAuthValidateFailure_MissingClientCredentials(t *testing.T) {
	var o oAuth2Config
	o.ClientSecret = "DEMOCLIENTSECRET"
	o.User = "salesforce_user"
	o.Password = "P@$$w0₹D"
	o.TokenURL = "https://login.salesforce.com/services/oauth2/token"

	assert.Error(t, o.Validate())
}

func TestConfigAuthValidateFailure_MissingUsernamePassword(t *testing.T) {
	var o oAuth2Config
	o.ClientID = "DEMOCLIENTID"
	o.ClientSecret = "DEMOCLIENTSECRET"
	o.TokenURL = "https://login.salesforce.com/services/oauth2/token"

	assert.Error(t, o.Validate())
}
