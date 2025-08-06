// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package okta

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// oktaTokenSource is a custom implementation of the oauth2.TokenSource interface.
// For more information, see https://pkg.go.dev/golang.org/x/oauth2#TokenSource.
type oktaTokenSource struct {
	mu      sync.Mutex
	ctx     context.Context
	conf    *oauth2.Config
	token   *oauth2.Token
	oktaJWK []byte
}

// clientSecretTokenSource is a custom implementation of the oauth2.TokenSource interface
// for client secret authentication.
type clientSecretTokenSource struct {
	mu           sync.Mutex
	ctx          context.Context
	conf         *oauth2.Config
	clientSecret string
	token        *oauth2.Token
}

// fetchOktaOauthClient creates an OAuth2 HTTP client for Okta authentication.
func (o *oAuth2Config) fetchOktaOauthClient(ctx context.Context, _ *http.Client) (*http.Client, error) {
	oauthConfig := &oauth2.Config{
		ClientID: o.ClientID,
		Scopes:   o.Scopes,
		Endpoint: oauth2.Endpoint{
			TokenURL: o.TokenURL,
		},
	}

	var tokenSource oauth2.TokenSource
	var err error

	// Determine authentication method based on provided credentials
	hasClientSecret := o.ClientSecret != ""
	hasJWTKeys := o.OktaJWKFile != "" || o.OktaJWKJSON != nil || o.OktaJWKPEM != nil

	if hasClientSecret {
		// Use client secret authentication
		tokenSource = &clientSecretTokenSource{
			ctx:          ctx,
			conf:         oauthConfig,
			clientSecret: o.ClientSecret,
		}
	} else if hasJWTKeys {
		// Use JWT-based authentication
		var oktaJWT string
		if o.OktaJWKFile != "" {
			oktaJWK, err := os.ReadFile(o.OktaJWKFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read JWK file: %w", err)
			}
			oktaJWT, err = generateOktaJWT(oktaJWK, oauthConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to generate Okta JWT: %w", err)
			}
		} else if o.OktaJWKJSON != nil {
			oktaJWT, err = generateOktaJWT(o.OktaJWKJSON, oauthConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to generate Okta JWT: %w", err)
			}
		} else if o.OktaJWKPEM != nil {
			oktaJWT, err = generateOktaJWTPEM(string(o.OktaJWKPEM), oauthConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to generate Okta JWT: %w", err)
			}
		} else {
			return nil, errors.New("no JWT credentials provided")
		}

		// Exchange JWT for bearer token
		token, err := exchangeForBearerToken(ctx, oktaJWT, oauthConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to exchange JWT for bearer token: %w", err)
		}

		tokenSource = &oktaTokenSource{
			ctx:     ctx,
			conf:    oauthConfig,
			oktaJWK: o.OktaJWKJSON,
			token:   token,
		}
	} else {
		return nil, errors.New("no authentication credentials provided")
	}

	return oauth2.NewClient(ctx, tokenSource), nil
}

// Token implements oauth2.TokenSource for client secret authentication.
func (cs *clientSecretTokenSource) Token() (*oauth2.Token, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Check if we have a valid token
	if cs.token != nil && cs.token.Valid() {
		return cs.token, nil
	}

	// Exchange client credentials for token
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("scope", strings.Join(cs.conf.Scopes, " "))
	data.Set("client_id", cs.conf.ClientID)
	data.Set("client_secret", cs.clientSecret)

	req, err := http.NewRequestWithContext(cs.ctx, "POST", cs.conf.Endpoint.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange client credentials: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResponse struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	cs.token = &oauth2.Token{
		AccessToken: tokenResponse.AccessToken,
		TokenType:   tokenResponse.TokenType,
		Expiry:      time.Now().Add(time.Duration(tokenResponse.ExpiresIn) * time.Second),
	}

	return cs.token, nil
}

// Token implements the oauth2.TokenSource interface and helps to implement
// custom token refresh logic. The parent context is passed via the
// customTokenSource struct since we cannot modify the function signature here.
func (ts *oktaTokenSource) Token() (*oauth2.Token, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	oktaJWT, err := generateOktaJWT(ts.oktaJWK, ts.conf)
	if err != nil {
		return nil, fmt.Errorf("error generating Okta JWT: %w", err)
	}
	token, err := exchangeForBearerToken(ts.ctx, oktaJWT, ts.conf)
	if err != nil {
		return nil, fmt.Errorf("error exchanging Okta JWT for bearer token: %w", err)
	}

	return token, nil
}

func generateOktaJWT(oktaJWK []byte, cnf *oauth2.Config) (string, error) {
	// Unmarshal the JWK into big ints.
	var jwkData struct {
		N    base64int `json:"n"`
		E    base64int `json:"e"`
		D    base64int `json:"d"`
		P    base64int `json:"p"`
		Q    base64int `json:"q"`
		Dp   base64int `json:"dp"`
		Dq   base64int `json:"dq"`
		Qinv base64int `json:"qi"`
	}
	err := json.Unmarshal(oktaJWK, &jwkData)
	if err != nil {
		return "", fmt.Errorf("error decoding JWK: %w", err)
	}

	// Create an RSA private key from JWK components.
	key := &rsa.PrivateKey{
		PublicKey: rsa.PublicKey{
			N: &jwkData.N.Int,
			E: int(jwkData.E.Int64()),
		},
		D:      &jwkData.D.Int,
		Primes: []*big.Int{&jwkData.P.Int, &jwkData.Q.Int},
		Precomputed: rsa.PrecomputedValues{
			Dp:   &jwkData.Dp.Int,
			Dq:   &jwkData.Dq.Int,
			Qinv: &jwkData.Qinv.Int,
		},
	}

	return signJWT(cnf, key)
}

// base64int is a JSON decoding shim for base64-encoded big.Int.
type base64int struct {
	big.Int
}

func (i *base64int) UnmarshalJSON(b []byte) error {
	src, ok := bytes.CutPrefix(b, []byte{'"'})
	if !ok {
		return fmt.Errorf("invalid JSON type: %s", b)
	}
	src, ok = bytes.CutSuffix(src, []byte{'"'})
	if !ok {
		return fmt.Errorf("invalid JSON type: %s", b)
	}
	dst := make([]byte, base64.RawURLEncoding.DecodedLen(len(src)))
	_, err := base64.RawURLEncoding.Decode(dst, src)
	if err != nil {
		return err
	}
	i.SetBytes(dst)
	return nil
}

func generateOktaJWTPEM(pemdata string, cnf *oauth2.Config) (string, error) {
	key, err := pemPKCS8PrivateKey([]byte(pemdata))
	if err != nil {
		return "", err
	}
	return signJWT(cnf, key)
}

// signJWT creates a JWT token using required claims and sign it with the
// private key.
func signJWT(cnf *oauth2.Config, key any) (string, error) {
	now := time.Now()
	signed, err := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.RegisteredClaims{
		Audience:  []string{cnf.Endpoint.TokenURL},
		Issuer:    cnf.ClientID,
		Subject:   cnf.ClientID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
	}).SignedString(key)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return signed, nil
}

// exchangeForBearerToken exchanges the Okta JWT for a bearer token.
func exchangeForBearerToken(ctx context.Context, oktaJWT string, cnf *oauth2.Config) (*oauth2.Token, error) {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("scope", strings.Join(cnf.Scopes, " "))
	data.Set("client_assertion_type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
	data.Set("client_assertion", oktaJWT)
	oauthConfig := &clientcredentials.Config{
		TokenURL:       cnf.Endpoint.TokenURL,
		EndpointParams: data,
	}
	tokenSource := oauthConfig.TokenSource(ctx)

	// get the access token
	accessToken, err := tokenSource.Token()
	if err != nil {
		return nil, err
	}

	return accessToken, nil
}
