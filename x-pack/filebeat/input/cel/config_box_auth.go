// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cel

import (
	"bytes"
	"context"
	"crypto"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/youmark/pkcs8"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/internal/dpop"
)

// boxTokenURL is the Box OAuth2 token endpoint.
const boxTokenURL = "https://api.box.com/oauth2/token" //nolint:gosec // G101: not a credential

// boxTokenSource is a custom oauth2.TokenSource that re-mints a short-lived
// JWT assertion each time the access token expires. Box does not issue refresh
// tokens for the JWT bearer grant, so a new assertion must be signed and
// exchanged on every refresh.
type boxTokenSource struct {
	mu    sync.Mutex
	ctx   context.Context
	creds boxCredentials
	token *oauth2.Token
}

// fetchBoxOauthClient builds an *http.Client whose transport automatically
// refreshes the Box access token by signing a new JWT assertion.
func (o *oAuth2Config) fetchBoxOauthClient(ctx context.Context) (*http.Client, error) {
	creds, err := o.resolveBoxCredentials()
	if err != nil {
		return nil, fmt.Errorf("oauth2 client: box credentials: %w", err)
	}

	token, err := exchangeBoxAssertion(ctx, creds, time.Now())
	if err != nil {
		return nil, fmt.Errorf("oauth2 client: box initial token exchange: %w", err)
	}

	ts := &boxTokenSource{
		ctx:   ctx,
		creds: creds,
		token: token,
	}
	return oauth2.NewClient(ctx, oauth2.ReuseTokenSource(token, ts)), nil
}

// Token implements oauth2.TokenSource.
func (ts *boxTokenSource) Token() (*oauth2.Token, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.token != nil && ts.token.Valid() {
		return ts.token, nil
	}

	token, err := exchangeBoxAssertion(ts.ctx, ts.creds, time.Now())
	if err != nil {
		return nil, fmt.Errorf("box token refresh: %w", err)
	}
	ts.token = token
	return token, nil
}

// boxCredentials holds the resolved, ready-to-use form of the Box app
// settings. It is separated from oAuth2Config so that validation and
// the runtime token loop share one parsing path.
type boxCredentials struct {
	clientID     string
	clientSecret string
	publicKeyID  string
	subjectType  string // "enterprise" or "user"
	subjectID    string
	tokenURL     string
	key          crypto.Signer
	extraParams  url.Values
}

// resolveBoxCredentials is the single resolution point for Box credentials.
// It is called by both validateBoxProvider (at startup) and
// fetchBoxOauthClient (at runtime).
func (o *oAuth2Config) resolveBoxCredentials() (boxCredentials, error) {
	var c boxCredentials

	if len(o.BoxConfigJSON) != 0 {
		var app boxAppConfig
		if err := json.Unmarshal(o.BoxConfigJSON, &app); err != nil {
			return c, fmt.Errorf("box.config_json: %w", err)
		}
		c.clientID = app.BoxAppSettings.ClientID
		c.clientSecret = app.BoxAppSettings.ClientSecret
		c.publicKeyID = app.BoxAppSettings.AppAuth.PublicKeyID

		subType := o.BoxSubjectType
		if subType == "" {
			subType = "enterprise"
		}
		c.subjectType = subType
		if subType == "enterprise" {
			subID := o.BoxSubjectID
			if subID == "" {
				subID = app.EnterpriseID
			}
			c.subjectID = subID
		} else {
			c.subjectID = o.BoxSubjectID
		}

		key, err := pemPrivateKey([]byte(app.BoxAppSettings.AppAuth.PrivateKey), app.BoxAppSettings.AppAuth.Passphrase)
		if err != nil {
			return c, fmt.Errorf("box.config_json private key: %w", err)
		}
		c.key = key
	} else {
		c.clientID = o.ClientID
		c.clientSecret = maybeString(o.ClientSecret)
		c.publicKeyID = o.BoxPublicKeyID

		subType := o.BoxSubjectType
		if subType == "" {
			subType = "enterprise"
		}
		c.subjectType = subType
		if subType == "enterprise" {
			subID := o.BoxSubjectID
			if subID == "" {
				subID = o.BoxEnterpriseID
			}
			c.subjectID = subID
		} else {
			c.subjectID = o.BoxSubjectID
		}

		key, err := pemPrivateKey([]byte(o.BoxPrivateKey), o.BoxPassphrase)
		if err != nil {
			return c, fmt.Errorf("box.private_key: %w", err)
		}
		c.key = key
	}

	c.tokenURL = o.getTokenURL()
	c.extraParams = o.EndpointParams
	return c, nil
}

// boxAppConfig is the typed shape of the Developer Console config.json file.
type boxAppConfig struct {
	BoxAppSettings struct {
		ClientID     string `json:"clientID"`
		ClientSecret string `json:"clientSecret"`
		AppAuth      struct {
			PublicKeyID string `json:"publicKeyID"`
			PrivateKey  string `json:"privateKey"`
			Passphrase  string `json:"passphrase"`
		} `json:"appAuth"`
	} `json:"boxAppSettings"`
	EnterpriseID string `json:"enterpriseID"`
}

// pemPrivateKey decodes a PEM block and parses it as a PKCS#8 private key.
// If pass is non-empty and the block type is "ENCRYPTED PRIVATE KEY" the block
// is decrypted with the passphrase before parsing.
func pemPrivateKey(pemdata []byte, pass string) (crypto.Signer, error) {
	blk, rest := pem.Decode(pemdata)
	if blk == nil {
		return nil, errors.New("no PEM data")
	}
	if rest := bytes.TrimSpace(rest); len(rest) != 0 {
		return nil, fmt.Errorf("PEM text has trailing data: %d bytes", len(rest))
	}

	var (
		key any
		err error
	)
	if blk.Type == "ENCRYPTED PRIVATE KEY" {
		key, err = pkcs8.ParsePKCS8PrivateKey(blk.Bytes, []byte(pass))
		if err != nil {
			return nil, fmt.Errorf("decrypting private key: %w", err)
		}
	} else {
		key, err = x509.ParsePKCS8PrivateKey(blk.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parsing private key: %w", err)
		}
	}

	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, fmt.Errorf("key is not a signer: %T", key)
	}
	return signer, nil
}

// exchangeBoxAssertion signs a JWT assertion and exchanges it for an access
// token at the Box token endpoint. On an invalid_grant error that includes a
// parseable Date response header the function retries once using the server's
// time, accommodating a skewed local clock.
func exchangeBoxAssertion(ctx context.Context, c boxCredentials, now time.Time) (*oauth2.Token, error) {
	tok, err := doExchangeBoxAssertion(ctx, c, now)
	if err != nil {
		// Retry once on invalid_grant, using the server's clock if available.
		var rerr *oauth2.RetrieveError
		if errors.As(err, &rerr) && rerr.ErrorCode == "invalid_grant" && rerr.Response != nil {
			serverTime, parseErr := http.ParseTime(rerr.Response.Header.Get("Date"))
			if parseErr == nil && !serverTime.IsZero() {
				tok, err = doExchangeBoxAssertion(ctx, c, serverTime)
			}
		}
	}
	return tok, err
}

func doExchangeBoxAssertion(ctx context.Context, c boxCredentials, now time.Time) (*oauth2.Token, error) {
	assertion, err := signBoxAssertion(c, now)
	if err != nil {
		return nil, err
	}

	ep := url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  {assertion},
	}
	maps.Copy(ep, c.extraParams)

	cfg := clientcredentials.Config{
		ClientID:       c.clientID,
		ClientSecret:   c.clientSecret,
		TokenURL:       c.tokenURL,
		AuthStyle:      oauth2.AuthStyleInParams,
		EndpointParams: ep,
	}
	return cfg.TokenSource(ctx).Token()
}

// boxClaims extends jwt.RegisteredClaims with the Box-specific box_sub_type
// private claim, which identifies whether the subject is an enterprise or user.
type boxClaims struct {
	jwt.RegisteredClaims
	SubType string `json:"box_sub_type"`
}

// signBoxAssertion creates and signs a JWT assertion for the Box JWT bearer
// grant. The assertion is valid for 45 seconds from now; Box allows at most
// 60 seconds so this leaves a margin for clock skew.
func signBoxAssertion(c boxCredentials, now time.Time) (string, error) {
	claims := boxClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    c.clientID,
			Subject:   c.subjectID,
			Audience:  []string{c.tokenURL},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(45 * time.Second)),
			ID:        dpop.RandomJTI(),
		},
		SubType: c.subjectType,
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	if c.publicKeyID != "" {
		tok.Header["kid"] = c.publicKeyID
	}

	signed, err := tok.SignedString(c.key)
	if err != nil {
		return "", fmt.Errorf("signing box assertion: %w", err)
	}
	return signed, nil
}

// validateBoxProvider checks that the Box provider configuration is coherent.
func (o *oAuth2Config) validateBoxProvider() error {
	// Box JWT bearer grant does not take a scope parameter.
	if len(o.Scopes) != 0 {
		return errors.New("box validation error: scopes must not be set for the Box JWT grant")
	}

	configJSONForm := len(o.BoxConfigJSON) != 0 || o.BoxConfigFile != ""
	discreteForm := o.BoxPrivateKey != ""

	if configJSONForm && discreteForm {
		return errors.New("box validation error: box.config_file/box.config_json and discrete fields are mutually exclusive")
	}
	if !configJSONForm && !discreteForm {
		return errors.New("box validation error: one of box.config_file, box.config_json or discrete credentials (box.private_key) must be provided")
	}

	if configJSONForm {
		if len(o.BoxConfigJSON) != 0 && o.BoxConfigFile != "" {
			return errors.New("box validation error: only one of box.config_file and box.config_json may be set")
		}
		if o.ClientID != "" || o.ClientSecret != nil || o.BoxEnterpriseID != "" ||
			o.BoxPublicKeyID != "" || o.BoxPassphrase != "" {
			return errors.New("box validation error: client.id, client.secret, box.enterprise_id, box.public_key_id and box.passphrase must not be set when using box.config_file or box.config_json")
		}
		// Promote from file so resolveBoxCredentials can parse it.
		if o.BoxConfigFile != "" {
			if err := populateJSONFromFile(o.BoxConfigFile, &o.BoxConfigJSON); err != nil {
				return fmt.Errorf("box validation error: %w", err)
			}
		}
	} else {
		if o.ClientID == "" || o.ClientSecret == nil {
			return errors.New("box validation error: client.id and client.secret must be provided when using discrete credentials")
		}
		if o.BoxConfigFile != "" || len(o.BoxConfigJSON) != 0 {
			return errors.New("box validation error: box.config_file and box.config_json must not be set when using discrete credentials")
		}
	}

	subType := o.BoxSubjectType
	if subType != "" && subType != "enterprise" && subType != "user" {
		return fmt.Errorf("box validation error: box.subject_type must be enterprise or user, got %q", subType)
	}
	if subType == "user" && o.BoxSubjectID == "" {
		return errors.New("box validation error: box.subject_id must be provided when box.subject_type is user")
	}
	if subType == "enterprise" && o.BoxSubjectID != "" {
		return errors.New("box validation error: box.subject_id must not be set when box.subject_type is enterprise")
	}

	// Reject any endpoint_params that would conflict with the reserved keys.
	for _, reserved := range []string{"grant_type", "assertion"} {
		if _, ok := o.EndpointParams[reserved]; ok {
			return fmt.Errorf("box validation error: endpoint_params must not include reserved key %q", reserved)
		}
	}

	// Eager key parse so a bad private key or wrong passphrase surfaces at startup.
	if _, err := o.resolveBoxCredentials(); err != nil {
		return fmt.Errorf("box validation error: %w", err)
	}

	return nil
}
