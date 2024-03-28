// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package salesforce

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	tests := map[string]struct {
		wantErr  error
		inputCfg config
	}{
		"no auth provider enabled (no password or jwt)": {
			inputCfg: config{
				Auth: &authConfig{
					OAuth2: &OAuth2{
						UserPasswordFlow: &UserPasswordFlow{},
						JWTBearerFlow:    &JWTBearerFlow{},
					},
				},
			},
			wantErr: errors.New("no auth provider enabled"),
		},
		"only one auth provider is allowed (either password or jwt)": {
			inputCfg: config{
				Auth: &authConfig{
					OAuth2: &OAuth2{
						UserPasswordFlow: &UserPasswordFlow{Enabled: pointer(true)},
						JWTBearerFlow:    &JWTBearerFlow{Enabled: pointer(true)},
					},
				},
			},
			wantErr: errors.New("only one auth provider must be enabled"),
		},
		"no instance url is configured (empty url)": {
			inputCfg: config{
				URL: "",
				Auth: &authConfig{
					OAuth2: &OAuth2{
						UserPasswordFlow: &UserPasswordFlow{Enabled: pointer(true)},
					},
				},
			},
			wantErr: errors.New("no instance url is configured"),
		},
		"no data collection method configured": {
			inputCfg: config{
				EventMonitoringMethod: &eventMonitoringMethod{},
				URL:                   "https://some-dummy-subdomain.salesforce.com/services/oauth2/token",
				Auth: &authConfig{
					OAuth2: &OAuth2{
						UserPasswordFlow: &UserPasswordFlow{Enabled: pointer(true)},
					},
				},
			},
			wantErr: errors.New(`at least one of "event_monitoring_method.event_log_file.enabled" or "event_monitoring_method.object.enabled" must be set to true`),
		},
		"invalid elf interval (1h)": {
			inputCfg: config{
				EventMonitoringMethod: &eventMonitoringMethod{
					EventLogFile: EventMonitoringConfig{
						Enabled:  pointer(true),
						Interval: time.Duration(0),
					},
				},
				URL: "https://some-dummy-subdomain.salesforce.com/services/oauth2/token",
				Auth: &authConfig{
					OAuth2: &OAuth2{
						UserPasswordFlow: &UserPasswordFlow{Enabled: pointer(true)},
					},
				},
			},
			wantErr: fmt.Errorf("not a valid interval %d", time.Duration(0)),
		},
		"invalid object interval (1h)": {
			inputCfg: config{
				EventMonitoringMethod: &eventMonitoringMethod{
					Object: EventMonitoringConfig{
						Enabled:  pointer(true),
						Interval: time.Duration(0),
					},
				},
				URL: "https://some-dummy-subdomain.salesforce.com/services/oauth2/token",
				Auth: &authConfig{
					OAuth2: &OAuth2{
						UserPasswordFlow: &UserPasswordFlow{Enabled: pointer(true)},
					},
				},
			},
			wantErr: fmt.Errorf("not a valid interval %d", time.Duration(0)),
		},
		"invalid api version (v45)": {
			inputCfg: config{
				Version: 45,
				EventMonitoringMethod: &eventMonitoringMethod{
					Object: EventMonitoringConfig{
						Enabled:  pointer(true),
						Interval: time.Hour,
					},
				},
				URL: "https://some-dummy-subdomain.salesforce.com/services/oauth2/token",
				Auth: &authConfig{
					OAuth2: &OAuth2{
						UserPasswordFlow: &UserPasswordFlow{Enabled: pointer(true)},
					},
				},
			},
			wantErr: errors.New("not a valid version i.e., 46.0 or above"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tc.inputCfg.Validate()
			assert.Equal(t, tc.wantErr, got)
		})
	}
}
