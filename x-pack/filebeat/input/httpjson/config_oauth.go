// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"golang.org/x/oauth2/endpoints"
	"golang.org/x/oauth2/google"
)

// An OAuth2Provider represents a supported oauth provider.
type OAuth2Provider string

const (
	OAuth2ProviderDefault OAuth2Provider = ""       // OAuth2ProviderDefault means no specific provider is set.
	OAuth2ProviderAzure   OAuth2Provider = "azure"  // OAuth2ProviderAzure AzureAD.
	OAuth2ProviderGoogle  OAuth2Provider = "google" // OAuth2ProviderGoogle Google.
)

func (p *OAuth2Provider) Unpack(in string) error {
	*p = OAuth2Provider(in)
	return nil
}

func (p OAuth2Provider) canonical() OAuth2Provider {
	return OAuth2Provider(strings.ToLower(string(p)))
}

// OAuth2 contains information about oauth2 authentication settings.
type OAuth2 struct {
	// common oauth fields
	ClientID       string              `config:"client.id"`
	ClientSecret   string              `config:"client.secret"`
	Enabled        *bool               `config:"enabled"`
	EndpointParams map[string][]string `config:"endpoint_params"`
	Provider       OAuth2Provider      `config:"provider"`
	Scopes         []string            `config:"scopes"`
	TokenURL       string              `config:"token_url"`

	// google specific
	GoogleCredentialsFile  string `config:"google.credentials_file"`
	GoogleCredentialsJSON  []byte `config:"google.credentials_json"`
	GoogleJWTFile          string `config:"google.jwt_file"`
	GoogleDelegatedAccount string `config:"google.delegated_account"`

	// microsoft azure specific
	AzureTenantID string `config:"azure.tenant_id"`
	AzureResource string `config:"azure.resource"`
}

// IsEnabled returns true if the `enable` field is set to true in the yaml.
func (o *OAuth2) IsEnabled() bool {
	return o != nil && (o.Enabled == nil || *o.Enabled)
}

// Client wraps the given http.Client and returns a new one that will use the oauth authentication.
func (o *OAuth2) Client(ctx context.Context, client *http.Client) (*http.Client, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, client)

	switch o.GetProvider() {
	case OAuth2ProviderAzure, OAuth2ProviderDefault:
		creds := clientcredentials.Config{
			ClientID:       o.ClientID,
			ClientSecret:   o.ClientSecret,
			TokenURL:       o.GetTokenURL(),
			Scopes:         o.Scopes,
			EndpointParams: o.GetEndpointParams(),
		}
		return creds.Client(ctx), nil
	case OAuth2ProviderGoogle:
		if o.GoogleJWTFile != "" {
			cfg, err := google.JWTConfigFromJSON(o.GoogleCredentialsJSON, o.Scopes...)
			if err != nil {
				return nil, fmt.Errorf("oauth2 client: error loading jwt credentials: %w", err)
			}
			cfg.Subject = o.GoogleDelegatedAccount
			return cfg.Client(ctx), nil
		}

		creds, err := google.CredentialsFromJSON(ctx, o.GoogleCredentialsJSON, o.Scopes...)
		if err != nil {
			return nil, fmt.Errorf("oauth2 client: error loading credentials: %w", err)
		}
		return oauth2.NewClient(ctx, creds.TokenSource), nil
	default:
		return nil, errors.New("oauth2 client: unknown provider")
	}
}

// GetTokenURL returns the TokenURL.
func (o *OAuth2) GetTokenURL() string {
	switch o.GetProvider() {
	case OAuth2ProviderAzure:
		if o.TokenURL == "" {
			return endpoints.AzureAD(o.AzureTenantID).TokenURL
		}
	}

	return o.TokenURL
}

// GetProvider returns provider in its canonical form.
func (o OAuth2) GetProvider() OAuth2Provider {
	return o.Provider.canonical()
}

// GetEndpointParams returns endpoint params with any provider ones combined.
func (o OAuth2) GetEndpointParams() map[string][]string {
	switch o.GetProvider() {
	case OAuth2ProviderAzure:
		if o.AzureResource != "" {
			if o.EndpointParams == nil {
				o.EndpointParams = map[string][]string{}
			}
			o.EndpointParams["resource"] = []string{o.AzureResource}
		}
	}

	return o.EndpointParams
}

// Validate checks if oauth2 config is valid.
func (o *OAuth2) Validate() error {
	switch o.GetProvider() {
	case OAuth2ProviderAzure:
		return o.validateAzureProvider()
	case OAuth2ProviderGoogle:
		return o.validateGoogleProvider()
	case OAuth2ProviderDefault:
		if o.TokenURL == "" || o.ClientID == "" || o.ClientSecret == "" {
			return errors.New("invalid configuration: both token_url and client credentials must be provided")
		}
	default:
		return fmt.Errorf("invalid configuration: unknown provider %q", o.GetProvider())
	}
	return nil
}

// findDefaultGoogleCredentials will default to google.FindDefaultCredentials and will only be changed for testing purposes
var findDefaultGoogleCredentials = google.FindDefaultCredentials

func (o *OAuth2) validateGoogleProvider() error {
	if o.TokenURL != "" || o.ClientID != "" || o.ClientSecret != "" ||
		o.AzureTenantID != "" || o.AzureResource != "" || len(o.EndpointParams) > 0 {
		return errors.New("invalid configuration: none of token_url and client credentials can be used, use google.credentials_file, google.jwt_file, google.credentials_json or ADC instead")
	}

	// credentials_json
	if len(o.GoogleCredentialsJSON) > 0 {
		if o.GoogleDelegatedAccount != "" {
			return errors.New("invalid configuration: google.delegated_account can only be provided with a jwt_file")
		}
		if !json.Valid(o.GoogleCredentialsJSON) {
			return errors.New("invalid configuration: google.credentials_json must be valid JSON")
		}
		return nil
	}

	// credentials_file
	if o.GoogleCredentialsFile != "" {
		if o.GoogleDelegatedAccount != "" {
			return errors.New("invalid configuration: google.delegated_account can only be provided with a jwt_file")
		}
		return o.populateCredentialsJSONFromFile(o.GoogleCredentialsFile)
	}

	// jwt_file
	if o.GoogleJWTFile != "" {
		return o.populateCredentialsJSONFromFile(o.GoogleJWTFile)
	}

	// Application Default Credentials (ADC)
	ctx := context.Background()
	if creds, err := findDefaultGoogleCredentials(ctx, o.Scopes...); err == nil {
		o.GoogleCredentialsJSON = creds.JSON
		return nil
	}

	return fmt.Errorf("invalid configuration: no authentication credentials were configured or detected (ADC)")
}

func (o *OAuth2) populateCredentialsJSONFromFile(file string) error {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return fmt.Errorf("invalid configuration: the file %q cannot be found", file)
	}

	credBytes, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("invalid configuration: the file %q cannot be read", file)
	}

	if !json.Valid(credBytes) {
		return fmt.Errorf("invalid configuration: the file %q does not contain valid JSON", file)
	}

	o.GoogleCredentialsJSON = credBytes

	return nil
}

func (o *OAuth2) validateAzureProvider() error {
	if o.TokenURL == "" && o.AzureTenantID == "" {
		return errors.New("invalid configuration: at least one of token_url or tenant_id must be provided")
	}
	if o.TokenURL != "" && o.AzureTenantID != "" {
		return errors.New("invalid configuration: only one of token_url and tenant_id can be used")
	}
	if o.ClientID == "" || o.ClientSecret == "" {
		return errors.New("invalid configuration: client credentials must be provided")
	}

	return nil
}
