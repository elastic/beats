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
		wantErr error
		config  UserPasswordFlow
	}{
		"auth disabled I":      {config: UserPasswordFlow{}, wantErr: nil},
		"auth disabled II":     {config: UserPasswordFlow{Enabled: pointer(false)}, wantErr: nil},
		"tokenURL missing":     {config: UserPasswordFlow{Enabled: pointer(true), TokenURL: ""}, wantErr: errors.New("token_url must be provided")},
		"clientID missing":     {config: UserPasswordFlow{Enabled: pointer(true), TokenURL: "https://salesforce.com", ClientID: ""}, wantErr: errors.New("client.id must be provided")},
		"clientSecret missing": {config: UserPasswordFlow{Enabled: pointer(true), TokenURL: "https://salesforce.com", ClientID: "xyz", ClientSecret: ""}, wantErr: errors.New("client.secret must be provided")},
		"username missing":     {config: UserPasswordFlow{Enabled: pointer(true), TokenURL: "https://salesforce.com", ClientID: "xyz", ClientSecret: "abc", Username: ""}, wantErr: errors.New("username must be provided")},
		"password missing":     {config: UserPasswordFlow{Enabled: pointer(true), TokenURL: "https://salesforce.com", ClientID: "xyz", ClientSecret: "abc", Username: "user", Password: ""}, wantErr: errors.New("password must be provided")},
		"all present":          {config: UserPasswordFlow{Enabled: pointer(true), TokenURL: "https://salesforce.com", ClientID: "xyz", ClientSecret: "abc", Username: "user", Password: "pass"}, wantErr: nil},
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
		wantErr error
		config  JWTBearerFlow
	}{
		"auth disabled I":        {config: JWTBearerFlow{}, wantErr: nil},
		"auth disabled II":       {config: JWTBearerFlow{Enabled: pointer(false)}, wantErr: nil},
		"url missing":            {config: JWTBearerFlow{Enabled: pointer(true), URL: ""}, wantErr: errors.New("url must be provided")},
		"clientID missing":       {config: JWTBearerFlow{Enabled: pointer(true), URL: "https://salesforce.com", ClientID: ""}, wantErr: errors.New("client.id must be provided")},
		"clientUsername missing": {config: JWTBearerFlow{Enabled: pointer(true), URL: "https://salesforce.com", ClientID: "xyz", ClientUsername: ""}, wantErr: errors.New("client.username must be provided")},
		"clientKeyPath missing":  {config: JWTBearerFlow{Enabled: pointer(true), URL: "https://salesforce.com", ClientID: "xyz", ClientUsername: "abc", ClientKeyPath: ""}, wantErr: errors.New("client.key_path must be provided")},
		"all present":            {config: JWTBearerFlow{Enabled: pointer(true), URL: "https://salesforce.com", ClientID: "xyz", ClientUsername: "abc", ClientKeyPath: "def"}, wantErr: nil},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tc.config.Validate()
			assert.Equal(t, tc.wantErr, got)
		})
	}
}
