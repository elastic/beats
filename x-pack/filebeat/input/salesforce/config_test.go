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
		"nil auth config": {
			inputCfg: config{
				EventMonitoringMethod: &eventMonitoringMethod{
					Object: EventMonitoringConfig{
						Enabled:  pointer(true),
						Interval: time.Hour,
					},
				},
				URL: "https://some-dummy-subdomain.salesforce.com/services/oauth2/token",
			},
			wantErr: errors.New("no auth provider enabled"),
		},
		"nil event monitoring config": {
			inputCfg: config{
				URL:     "https://some-dummy-subdomain.salesforce.com/services/oauth2/token",
				Version: 56,
				Auth: &authConfig{
					OAuth2: &OAuth2{
						UserPasswordFlow: &UserPasswordFlow{Enabled: pointer(true)},
					},
				},
			},
			wantErr: errors.New(`at least one of "event_monitoring_method.event_log_file.enabled" or "event_monitoring_method.object.enabled" must be set to true`),
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
		"missing object query config": {
			inputCfg: config{
				Version: 56,
				EventMonitoringMethod: &eventMonitoringMethod{
					Object: EventMonitoringConfig{
						Enabled:  pointer(true),
						Interval: time.Hour,
						Cursor:   &cursorConfig{Field: "EventDate"},
					},
				},
				URL: "https://some-dummy-subdomain.salesforce.com/services/oauth2/token",
				Auth: &authConfig{
					OAuth2: &OAuth2{
						UserPasswordFlow: &UserPasswordFlow{Enabled: pointer(true)},
					},
				},
			},
			wantErr: errors.New(`"event_monitoring_method.object.query" must be configured when "event_monitoring_method.object.enabled" is true`),
		},
		"missing object cursor field": {
			inputCfg: config{
				Version: 56,
				EventMonitoringMethod: &eventMonitoringMethod{
					Object: EventMonitoringConfig{
						Enabled:  pointer(true),
						Interval: time.Hour,
						Query: &QueryConfig{
							Default: &valueTpl{},
							Value:   &valueTpl{},
						},
						Cursor: &cursorConfig{},
					},
				},
				URL: "https://some-dummy-subdomain.salesforce.com/services/oauth2/token",
				Auth: &authConfig{
					OAuth2: &OAuth2{
						UserPasswordFlow: &UserPasswordFlow{Enabled: pointer(true)},
					},
				},
			},
			wantErr: errors.New(`"event_monitoring_method.object.cursor.field" must be configured when "event_monitoring_method.object.enabled" is true`),
		},
		"missing event log file query value": {
			inputCfg: config{
				Version: 56,
				EventMonitoringMethod: &eventMonitoringMethod{
					EventLogFile: EventMonitoringConfig{
						Enabled:  pointer(true),
						Interval: time.Hour,
						Query: &QueryConfig{
							Default: &valueTpl{},
						},
						Cursor: &cursorConfig{Field: "CreatedDate"},
					},
				},
				URL: "https://some-dummy-subdomain.salesforce.com/services/oauth2/token",
				Auth: &authConfig{
					OAuth2: &OAuth2{
						UserPasswordFlow: &UserPasswordFlow{Enabled: pointer(true)},
					},
				},
			},
			wantErr: errors.New(`"event_monitoring_method.event_log_file.query.value" must be configured when "event_monitoring_method.event_log_file.enabled" is true`),
		},
		"invalid api version (v45)": {
			inputCfg: config{
				Version: 45,
				EventMonitoringMethod: &eventMonitoringMethod{
					Object: EventMonitoringConfig{
						Enabled:  pointer(true),
						Interval: time.Hour,
						Query: &QueryConfig{
							Default: &valueTpl{},
							Value:   &valueTpl{},
						},
						Cursor: &cursorConfig{Field: "EventDate"},
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
		"invalid object batch initial interval": {
			inputCfg: config{
				Version: 56,
				EventMonitoringMethod: &eventMonitoringMethod{
					Object: EventMonitoringConfig{
						Enabled:  pointer(true),
						Interval: time.Hour,
						Batch: &batchConfig{
							Enabled:          pointer(true),
							InitialInterval:  0,
							MaxWindowsPerRun: pointer(1),
							Window:           5 * time.Minute,
						},
					},
				},
				URL: "https://some-dummy-subdomain.salesforce.com/services/oauth2/token",
				Auth: &authConfig{
					OAuth2: &OAuth2{
						UserPasswordFlow: &UserPasswordFlow{Enabled: pointer(true)},
					},
				},
			},
			wantErr: errors.New(`"event_monitoring_method.object.batch.initial_interval" must be greater than zero`),
		},
		"invalid object batch window": {
			inputCfg: config{
				Version: 56,
				EventMonitoringMethod: &eventMonitoringMethod{
					Object: EventMonitoringConfig{
						Enabled:  pointer(true),
						Interval: time.Hour,
						Batch: &batchConfig{
							Enabled:          pointer(true),
							InitialInterval:  time.Hour,
							MaxWindowsPerRun: pointer(1),
							Window:           0,
						},
					},
				},
				URL: "https://some-dummy-subdomain.salesforce.com/services/oauth2/token",
				Auth: &authConfig{
					OAuth2: &OAuth2{
						UserPasswordFlow: &UserPasswordFlow{Enabled: pointer(true)},
					},
				},
			},
			wantErr: errors.New(`"event_monitoring_method.object.batch.window" must be greater than zero`),
		},
		"invalid object batch max windows per run": {
			inputCfg: config{
				Version: 56,
				EventMonitoringMethod: &eventMonitoringMethod{
					Object: EventMonitoringConfig{
						Enabled:  pointer(true),
						Interval: time.Hour,
						Batch: &batchConfig{
							Enabled:          pointer(true),
							InitialInterval:  time.Hour,
							MaxWindowsPerRun: pointer(0),
							Window:           5 * time.Minute,
						},
					},
				},
				URL: "https://some-dummy-subdomain.salesforce.com/services/oauth2/token",
				Auth: &authConfig{
					OAuth2: &OAuth2{
						UserPasswordFlow: &UserPasswordFlow{Enabled: pointer(true)},
					},
				},
			},
			wantErr: errors.New(`"event_monitoring_method.object.batch.max_windows_per_run" must be greater than zero`),
		},
		"object batching requires batch-aware query window placeholders": {
			inputCfg: config{
				Version: 56,
				EventMonitoringMethod: &eventMonitoringMethod{
					Object: EventMonitoringConfig{
						Enabled:  pointer(true),
						Interval: time.Hour,
						Batch: &batchConfig{
							Enabled:          pointer(true),
							InitialInterval:  time.Hour,
							MaxWindowsPerRun: pointer(2),
							Window:           5 * time.Minute,
						},
						Query: &QueryConfig{
							Default: getValueTpl(defaultLoginObjectQuery),
							Value:   getValueTpl(valueLoginObjectQuery),
						},
						Cursor: &cursorConfig{Field: "EventDate"},
					},
				},
				URL: "https://some-dummy-subdomain.salesforce.com/services/oauth2/token",
				Auth: &authConfig{
					OAuth2: &OAuth2{
						UserPasswordFlow: &UserPasswordFlow{Enabled: pointer(true)},
					},
				},
			},
			wantErr: errors.New(`"event_monitoring_method.object.query.value" must reference both ".cursor.object.batch_start_time" and ".cursor.object.batch_end_time" when "event_monitoring_method.object.batch.enabled" is true`),
		},
		"unbatched object query must not reference batch window placeholders": {
			inputCfg: config{
				Version: 56,
				EventMonitoringMethod: &eventMonitoringMethod{
					Object: EventMonitoringConfig{
						Enabled:  pointer(true),
						Interval: time.Hour,
						Query: &QueryConfig{
							Default: getValueTpl(defaultLoginObjectQuery),
							Value:   getValueTpl(valueBatchedLoginObjectQuery),
						},
						Cursor: &cursorConfig{Field: "EventDate"},
					},
				},
				URL: "https://some-dummy-subdomain.salesforce.com/services/oauth2/token",
				Auth: &authConfig{
					OAuth2: &OAuth2{
						UserPasswordFlow: &UserPasswordFlow{Enabled: pointer(true)},
					},
				},
			},
			wantErr: errors.New(`"event_monitoring_method.object.query.value" must not reference ".cursor.object.batch_start_time" or ".cursor.object.batch_end_time" when "event_monitoring_method.object.batch.enabled" is false`),
		},
		"valid object batch config": {
			inputCfg: config{
				Version: 56,
				EventMonitoringMethod: &eventMonitoringMethod{
					Object: EventMonitoringConfig{
						Enabled:  pointer(true),
						Interval: time.Hour,
						Batch: &batchConfig{
							Enabled:          pointer(true),
							InitialInterval:  time.Hour,
							MaxWindowsPerRun: pointer(2),
							Window:           5 * time.Minute,
						},
						Query: &QueryConfig{
							Default: getValueTpl(defaultLoginObjectQuery),
							Value:   getValueTpl(valueBatchedLoginObjectQuery),
						},
						Cursor: &cursorConfig{Field: "EventDate"},
					},
				},
				URL: "https://some-dummy-subdomain.salesforce.com/services/oauth2/token",
				Auth: &authConfig{
					OAuth2: &OAuth2{
						UserPasswordFlow: &UserPasswordFlow{Enabled: pointer(true)},
					},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tc.inputCfg.Validate()
			assert.Equal(t, tc.wantErr, got)
		})
	}
}
