// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package o365audit

import (
	"fmt"
	"time"

	"github.com/elastic/beats/x-pack/filebeat/input/o365audit/auth"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common/transport/tlscommon"
)

// Config for the O365 audit API input.
type Config struct {
	// CertificateConfig contains the authentication credentials (certificate).
	CertificateConfig tlscommon.CertificateConfig `config:",inline"`

	// ApplicationID (aka. client ID) of the Azure application.
	ApplicationID string `config:"application_id" validate:"required"`

	ClientSecret string `config:"client_secret"`

	// TenantID (aka. Directory ID) is a list of tenants for which to fetch
	// the audit logs. This can be a string or a list of strings.
	TenantID interface{} `config:"tenant_id,replace" validate:"required"`

	// Content-Type is a list of content-types to fetch.
	// This can be a string or a list of strings.
	ContentType interface{} `config:"content_type,replace"`

	// API contains settings to adapt to changes on the API.
	API APIConfig `config:"api"`

	tenants      []string
	contentTypes []string
}

// APIConfig contains advanced settings that are only supposed to be changed
// to diagnose errors or to adapt to changes in the service.
type APIConfig struct {

	// AuthenticationEndpoint to authorize the Azure app.
	AuthenticationEndpoint string `config:"authentication_endpoint"`

	// Resource to request authorization for.
	Resource string `config:"resource"`

	// MaxRetention determines how far back the input will poll for events.
	MaxRetention time.Duration `config:"max_retention" validate:"positive"`

	// AdjustClock controls whether the input will adapt its internal clock
	// to the server's clock to compensate for clock differences when the API
	// returns an error indicating that the times requests are out of bounds.
	AdjustClock bool `config:"adjust_clock"`

	// AdjustClockMinDifference sets the minimum difference between clocks so
	// that an adjust is considered.
	AdjustClockMinDifference time.Duration `config:"adjust_clock_min_difference" validate:"positive"`

	// AdjustClockWarn controls whether a warning should be printed to the logs
	// when a clock difference between the local clock and the server's clock
	// is detected, as it can lead to event loss.
	AdjustClockWarn bool `config:"adjust_clock_warn"`

	// ErrorRetryInterval sets the interval between retries in the case of
	// errors performing a request.
	ErrorRetryInterval time.Duration `config:"error_retry_interval" validate:"positive"`

	// LiveWindowSize defines the window of time [now-window, now) that will be
	// used to poll for new events. If events are created outside of this window,
	// they will be lost.
	LiveWindowSize time.Duration `config:"live_window_size" validate:"positive"`

	// LiveWindowPollInterval determines how often the input should poll for new
	// data once it has finished scanning for past events and reached the live
	// window.
	LiveWindowPollInterval time.Duration `config:"live_window_poll_interval" validate:"positive"`

	// MaxRequestsPerMinute sets the limit on the number of API requests that
	// can be sent, per tenant.
	MaxRequestsPerMinute int `config:"max_requests_per_minute" validate:"positive"`
}

func defaultConfig() Config {
	return Config{

		// All documented content types.
		ContentType: []string{
			"Audit.AzureActiveDirectory",
			"Audit.Exchange",
			"Audit.SharePoint",
			"Audit.General",
			"DLP.All",
		},

		API: APIConfig{
			// This is used to bootstrap the input for the first time
			// as the API doesn't provide a way to query for the oldest record.
			// Currently the API will err on queries older than this, use with care.
			MaxRetention: 7 * timeDay,

			AuthenticationEndpoint: "https://login.microsoftonline.com/",

			Resource: "https://manage.office.com",

			AdjustClock: true,

			AdjustClockMinDifference: 5 * time.Minute,

			AdjustClockWarn: true,

			ErrorRetryInterval: 5 * time.Minute,

			LiveWindowPollInterval: time.Minute,

			LiveWindowSize: timeDay,

			// According to the docs this is the max requests that are allowed
			// per tenant per minute.
			MaxRequestsPerMinute: 2000,
		},
	}
}

// Validate checks that the configuration is correct.
func (c *Config) Validate() (err error) {
	hasSecret := c.ClientSecret != ""
	hasCert := c.CertificateConfig.Certificate != ""

	if !hasSecret && !hasCert {
		return errors.New("no authentication configured. Configure a client_secret or a certificate and key.")
	}
	if hasSecret && hasCert {
		return errors.New("both client_secret and certificate are configured. Only one authentication method can be used.")
	}
	if hasCert {
		if err = c.CertificateConfig.Validate(); err != nil {
			return errors.Wrap(err, "invalid certificate config")
		}
	}
	if c.tenants, err = asStringList(c.TenantID); err != nil {
		return errors.Wrap(err, "error validating tenant_id")
	}
	if c.contentTypes, err = asStringList(c.ContentType); err != nil {
		return errors.Wrap(err, "error validating content_type")
	}
	return nil
}

// A helper to allow defining a field either as a string or a list of strings.
func asStringList(value interface{}) (list []string, err error) {
	switch v := value.(type) {
	case string:
		list = []string{v}
	case []string:
		list = v
	case []interface{}:
		list = make([]string, len(v))
		for idx, ival := range v {
			str, ok := ival.(string)
			if !ok {
				return nil, fmt.Errorf("string value required. Found %v (type %T) at position %d",
					ival, ival, idx+1)
			}
			list[idx] = str
		}
	default:
		return nil, fmt.Errorf("array of strings required. Found %v (type %T)", value, value)
	}
	return list, nil
}

// NewTokenProvider returns an auth.TokenProvider for the given tenantID.
func (c *Config) NewTokenProvider(tenantID string) (auth.TokenProvider, error) {
	if c.ClientSecret != "" {
		return auth.NewProviderFromClientSecret(
			c.API.AuthenticationEndpoint,
			c.API.Resource,
			c.ApplicationID,
			tenantID,
			c.ClientSecret,
		)
	}
	return auth.NewProviderFromCertificate(
		c.API.AuthenticationEndpoint,
		c.API.Resource,
		c.ApplicationID,
		tenantID,
		c.CertificateConfig,
	)
}
