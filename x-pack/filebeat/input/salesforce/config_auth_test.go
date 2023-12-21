// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package salesforce

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOAuth2Config(t *testing.T) {
	tests := map[string]struct {
		config  oAuth2Config
		wantErr error
	}{
		"auth disabled I":      {config: oAuth2Config{}, wantErr: nil},
		"auth disabled II":     {config: oAuth2Config{Enabled: pointer(false)}, wantErr: nil},
		"tokenURL missing":     {config: oAuth2Config{Enabled: pointer(true), TokenURL: ""}, wantErr: errors.New("token_url must be provided")},
		"clientID missing":     {config: oAuth2Config{Enabled: pointer(true), TokenURL: "https://salesforce.com", ClientID: ""}, wantErr: errors.New("client.id must be provided")},
		"clientSecret missing": {config: oAuth2Config{Enabled: pointer(true), TokenURL: "https://salesforce.com", ClientID: "xyz", ClientSecret: ""}, wantErr: errors.New("client.secret must be provided")},
		"user missing":         {config: oAuth2Config{Enabled: pointer(true), TokenURL: "https://salesforce.com", ClientID: "xyz", ClientSecret: "abc", User: ""}, wantErr: errors.New("user must be provided")},
		"password missing":     {config: oAuth2Config{Enabled: pointer(true), TokenURL: "https://salesforce.com", ClientID: "xyz", ClientSecret: "abc", User: "user", Password: ""}, wantErr: errors.New("password must be provided")},
		"all present":          {config: oAuth2Config{Enabled: pointer(true), TokenURL: "https://salesforce.com", ClientID: "xyz", ClientSecret: "abc", User: "user", Password: "pass"}, wantErr: nil},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tc.config.Validate()
			assert.Equal(t, tc.wantErr, got)
		})
	}
}

func TestJWTConfig(t *testing.T) {
	tests := map[string]struct {
		config  jwtConfig
		wantErr error
	}{
		"auth disabled I":        {config: jwtConfig{}, wantErr: nil},
		"auth disabled II":       {config: jwtConfig{Enabled: pointer(false)}, wantErr: nil},
		"url missing":            {config: jwtConfig{Enabled: pointer(true), URL: ""}, wantErr: errors.New("url must be provided")},
		"clientID missing":       {config: jwtConfig{Enabled: pointer(true), URL: "https://salesforce.com", ClientID: ""}, wantErr: errors.New("client.id must be provided")},
		"clientUsername missing": {config: jwtConfig{Enabled: pointer(true), URL: "https://salesforce.com", ClientID: "xyz", ClientUsername: ""}, wantErr: errors.New("client.username must be provided")},
		"clientKeyPath missing":  {config: jwtConfig{Enabled: pointer(true), URL: "https://salesforce.com", ClientID: "xyz", ClientUsername: "abc", ClientKeyPath: ""}, wantErr: errors.New("client.key_path must be provided")},
		"all present":            {config: jwtConfig{Enabled: pointer(true), URL: "https://salesforce.com", ClientID: "xyz", ClientUsername: "abc", ClientKeyPath: "def"}, wantErr: nil},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tc.config.Validate()
			assert.Equal(t, tc.wantErr, got)
		})
	}
}
