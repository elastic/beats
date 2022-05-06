// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cometd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	demoClientID     = "DEMOCLIENTID"
	demoClientSecret = "DEMOCLIENTSECRET"
	salesforceUser   = "salesforce_user"
	pwd              = "P@$$w0₹D"                                           //nolint:gosec // Bad linter! The pwd is used as testing purpose.
	tokenURL         = "https://login.salesforce.com/services/oauth2/token" //nolint:gosec // Bad linter! The tokenURL is used as testing purpose.
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
	o.ClientID = demoClientID
	o.ClientSecret = demoClientSecret
	o.User = salesforceUser
	o.Password = pwd
	o.TokenURL = tokenURL

	assert.NoError(t, o.Validate())
}

func TestConfigAuthValidateFailure_MissingTokenURL(t *testing.T) {
	var o oAuth2Config
	o.ClientID = demoClientID
	o.ClientSecret = demoClientSecret
	o.User = salesforceUser
	o.Password = pwd

	assert.Error(t, o.Validate())
}

func TestConfigAuthValidateFailure_MissingClientCredentials(t *testing.T) {
	var o oAuth2Config
	o.ClientSecret = demoClientSecret
	o.User = salesforceUser
	o.Password = pwd
	o.TokenURL = tokenURL

	assert.Error(t, o.Validate())
}

func TestConfigAuthValidateFailure_MissingUsernamePassword(t *testing.T) {
	var o oAuth2Config
	o.ClientID = demoClientID
	o.ClientSecret = demoClientSecret
	o.TokenURL = tokenURL

	assert.Error(t, o.Validate())
}
