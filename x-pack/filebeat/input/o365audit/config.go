// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package o365audit

import (
	"fmt"
	"net/url"
	"time"

	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/menderesk/beats/v7/x-pack/filebeat/input/o365audit/auth"
)

// Config for the O365 audit API input.
type Config struct {
	// CertificateConfig contains the authentication credentials (certificate).
	CertificateConfig tlscommon.CertificateConfig `config:",inline"`

	// ApplicationID (aka. client ID) of the Azure application.
	ApplicationID string `config:"application_id" validate:"required"`

	// ClientSecret (aka. API key) to use for authentication.
	ClientSecret string `config:"client_secret"`

	// TenantID (aka. Directory ID) is a list of tenants for which to fetch
	// the audit logs. This can be a string or a list of strings.
	TenantID stringList `config:"tenant_id,replace" validate:"required"`

	// Content-Type is a list of content-types to fetch.
	// This can be a string or a list of strings.
	ContentType stringList `config:"content_type,replace"`

	// API contains settings to adapt to changes on the API.
	API APIConfig `config:"api"`
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

	// PollInterval determines how often the input should poll for new
	// data once it has finished scanning for past events and reached the live
	// window.
	PollInterval time.Duration `config:"poll_interval" validate:"positive"`

	// MaxRequestsPerMinute sets the limit on the number of API requests that
	// can be sent, per tenant.
	MaxRequestsPerMinute int `config:"max_requests_per_minute" validate:"positive"`

	// SetIDFromAuditRecord controls whether the unique "Id" field in audit
	// record is used as the document id for ingestion. This helps avoiding
	// duplicates.
	SetIDFromAuditRecord bool `config:"set_id_from_audit_record"`

	// PreserveOriginalEvent controls whether the original o365 audit object
	// will be kept in `event.original` or not.
	PreserveOriginalEvent bool `config:"preserve_original_event"`

	// MaxQuerySize is the maximum time window that can be queried. The default
	// is 24h.
	MaxQuerySize time.Duration `config:"max_query_size" validate:"positive"`
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

			PollInterval: 3 * time.Minute,

			MaxQuerySize: timeDay,

			// According to the docs this is the max requests that are allowed
			// per tenant per minute.
			MaxRequestsPerMinute: 2000,

			SetIDFromAuditRecord: true,
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
	c.API.Resource, err = forceURLScheme(c.API.Resource, "https")
	if err != nil {
		return errors.Wrapf(err, "resource '%s' is not a valid URL", c.API.Resource)
	}
	c.API.AuthenticationEndpoint, err = forceURLScheme(c.API.AuthenticationEndpoint, "https")
	if err != nil {
		return errors.Wrapf(err, "authentication_endpoint '%s' is not a valid URL", c.API.AuthenticationEndpoint)
	}
	return nil
}

type stringList []string

// Unpack populates the stringList with either a single string value or an array.
func (s *stringList) Unpack(value interface{}) error {
	switch v := value.(type) {
	case string:
		*s = []string{v}
	case []string:
		*s = v
	case []interface{}:
		*s = make([]string, len(v))
		for idx, ival := range v {
			str, ok := ival.(string)
			if !ok {
				return fmt.Errorf("string value required. Found %v (type %T) at position %d",
					ival, ival, idx+1)
			}
			(*s)[idx] = str
		}
	default:
		return fmt.Errorf("array of strings required. Found %v (type %T)", value, value)
	}
	return nil
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

// Ensures that the passed URL has a scheme, using the provided one if needed.
// Returns an error is the URL can't be parsed.
func forceURLScheme(baseURL, scheme string) (urlWithScheme string, err error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	// Scheme is mandatory
	if parsed.Scheme == "" {
		withResource := "https://" + baseURL
		if parsed, err = url.Parse(withResource); err != nil {
			return "", err
		}
	}
	return parsed.String(), nil
}
