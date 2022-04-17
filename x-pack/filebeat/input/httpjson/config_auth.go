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

	"github.com/menderesk/beats/v7/libbeat/common"
)

// authStyleInParams sends the "client_id" and "client_secret" in the POST body as application/x-www-form-urlencoded parameters.
const authStyleInParams = 1

type authConfig struct {
	Basic  *basicAuthConfig `config:"basic"`
	OAuth2 *oAuth2Config    `config:"oauth2"`
}

func (c authConfig) Validate() error {
	if c.Basic.isEnabled() && c.OAuth2.isEnabled() {
		return errors.New("only one kind of auth can be enabled")
	}
	return nil
}

type basicAuthConfig struct {
	Enabled  *bool  `config:"enabled"`
	User     string `config:"user"`
	Password string `config:"password"`
}

// IsEnabled returns true if the `enable` field is set to true in the yaml.
func (b *basicAuthConfig) isEnabled() bool {
	return b != nil && (b.Enabled == nil || *b.Enabled)
}

// Validate checks if oauth2 config is valid.
func (b *basicAuthConfig) Validate() error {
	if !b.isEnabled() {
		return nil
	}

	if b.User == "" || b.Password == "" {
		return errors.New("both user and password must be set")
	}

	return nil
}

// An oAuth2Provider represents a supported oauth provider.
type oAuth2Provider string

const (
	oAuth2ProviderDefault oAuth2Provider = ""       // oAuth2ProviderDefault means no specific provider is set.
	oAuth2ProviderAzure   oAuth2Provider = "azure"  // oAuth2ProviderAzure AzureAD.
	oAuth2ProviderGoogle  oAuth2Provider = "google" // oAuth2ProviderGoogle Google.
)

func (p *oAuth2Provider) Unpack(in string) error {
	*p = oAuth2Provider(in)
	return nil
}

func (p oAuth2Provider) canonical() oAuth2Provider {
	return oAuth2Provider(strings.ToLower(string(p)))
}

type oAuth2Config struct {
	Enabled *bool `config:"enabled"`

	// common oauth fields
	ClientID       string              `config:"client.id"`
	ClientSecret   string              `config:"client.secret"`
	EndpointParams map[string][]string `config:"endpoint_params"`
	Password       string              `config:"password"`
	Provider       oAuth2Provider      `config:"provider"`
	Scopes         []string            `config:"scopes"`
	TokenURL       string              `config:"token_url"`
	User           string              `config:"user"`

	// google specific
	GoogleCredentialsFile  string          `config:"google.credentials_file"`
	GoogleCredentialsJSON  common.JSONBlob `config:"google.credentials_json"`
	GoogleJWTFile          string          `config:"google.jwt_file"`
	GoogleDelegatedAccount string          `config:"google.delegated_account"`

	// microsoft azure specific
	AzureTenantID string `config:"azure.tenant_id"`
	AzureResource string `config:"azure.resource"`
}

// IsEnabled returns true if the `enable` field is set to true in the yaml.
func (o *oAuth2Config) isEnabled() bool {
	return o != nil && (o.Enabled == nil || *o.Enabled)
}

// clientCredentialsGrant creates http client from token_url and client credentials
// held by the receiver.
func (o *oAuth2Config) clientCredentialsGrant(ctx context.Context, _ *http.Client) *http.Client {
	creds := clientcredentials.Config{
		ClientID:       o.ClientID,
		ClientSecret:   o.ClientSecret,
		TokenURL:       o.getTokenURL(),
		Scopes:         o.Scopes,
		EndpointParams: o.getEndpointParams(),
	}
	return creds.Client(ctx)
}

// Client wraps the given http.Client and returns a new one that will use the oauth authentication.
func (o *oAuth2Config) client(ctx context.Context, client *http.Client) (*http.Client, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, client)

	switch o.getProvider() {
	case oAuth2ProviderDefault:
		if o.User != "" || o.Password != "" {
			conf := &oauth2.Config{
				ClientID:     o.ClientID,
				ClientSecret: o.ClientSecret,
				Endpoint: oauth2.Endpoint{
					TokenURL:  o.TokenURL,
					AuthStyle: authStyleInParams,
				},
			}
			token, err := conf.PasswordCredentialsToken(ctx, o.User, o.Password)
			if err != nil {
				return nil, fmt.Errorf("oauth2 client: error loading credentials using user and password: %w", err)
			}
			return conf.Client(ctx, token), nil
		} else {
			return o.clientCredentialsGrant(ctx, client), nil
		}
	case oAuth2ProviderAzure:
		return o.clientCredentialsGrant(ctx, client), nil
	case oAuth2ProviderGoogle:
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

// getTokenURL returns the TokenURL.
func (o *oAuth2Config) getTokenURL() string {
	switch o.getProvider() {
	case oAuth2ProviderAzure:
		if o.TokenURL == "" {
			return endpoints.AzureAD(o.AzureTenantID).TokenURL
		}
	}

	return o.TokenURL
}

// getProvider returns provider in its canonical form.
func (o oAuth2Config) getProvider() oAuth2Provider {
	return o.Provider.canonical()
}

// getEndpointParams returns endpoint params with any provider ones combined.
func (o oAuth2Config) getEndpointParams() map[string][]string {
	switch o.getProvider() {
	case oAuth2ProviderAzure:
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
func (o *oAuth2Config) Validate() error {
	if !o.isEnabled() {
		return nil
	}

	switch o.getProvider() {
	case oAuth2ProviderAzure:
		return o.validateAzureProvider()
	case oAuth2ProviderGoogle:
		return o.validateGoogleProvider()
	case oAuth2ProviderDefault:
		if o.TokenURL == "" || o.ClientID == "" || o.ClientSecret == "" {
			return errors.New("both token_url and client credentials must be provided")
		}
		if (o.User != "" && o.Password == "") || (o.User == "" && o.Password != "") {
			return errors.New("both user and password credentials must be provided")
		}
	default:
		return fmt.Errorf("unknown provider %q", o.getProvider())
	}

	return nil
}

// findDefaultGoogleCredentials will default to google.FindDefaultCredentials and will only be changed for testing purposes
var findDefaultGoogleCredentials = google.FindDefaultCredentials

func (o *oAuth2Config) validateGoogleProvider() error {
	if o.TokenURL != "" || o.ClientID != "" || o.ClientSecret != "" ||
		o.AzureTenantID != "" || o.AzureResource != "" || len(o.EndpointParams) > 0 {
		return errors.New("none of token_url and client credentials can be used, use google.credentials_file, google.jwt_file, google.credentials_json or ADC instead")
	}

	// credentials_json
	if len(o.GoogleCredentialsJSON) > 0 {
		if o.GoogleDelegatedAccount != "" {
			return errors.New("google.delegated_account can only be provided with a jwt_file")
		}
		return nil
	}

	// credentials_file
	if o.GoogleCredentialsFile != "" {
		if o.GoogleDelegatedAccount != "" {
			return errors.New("google.delegated_account can only be provided with a jwt_file")
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

	return fmt.Errorf("no authentication credentials were configured or detected (ADC)")
}

func (o *oAuth2Config) populateCredentialsJSONFromFile(file string) error {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return fmt.Errorf("the file %q cannot be found", file)
	}

	credBytes, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("the file %q cannot be read", file)
	}

	if !json.Valid(credBytes) {
		return fmt.Errorf("the file %q does not contain valid JSON", file)
	}

	o.GoogleCredentialsJSON = credBytes

	return nil
}

func (o *oAuth2Config) validateAzureProvider() error {
	if o.TokenURL == "" && o.AzureTenantID == "" {
		return errors.New("at least one of token_url or tenant_id must be provided")
	}
	if o.TokenURL != "" && o.AzureTenantID != "" {
		return errors.New("only one of token_url and tenant_id can be used")
	}
	if o.ClientID == "" || o.ClientSecret == "" {
		return errors.New("client credentials must be provided")
	}

	return nil
}
