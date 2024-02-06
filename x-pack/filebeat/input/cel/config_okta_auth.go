// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cel

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// oktaTokenSource is a custom implementation of the oauth2.TokenSource interface.
// For more information, see https://pkg.go.dev/golang.org/x/oauth2#TokenSource
type oktaTokenSource struct {
	mu      sync.Mutex
	ctx     context.Context
	conf    *oauth2.Config
	token   *oauth2.Token
	oktaJWK []byte
}

// fetchOktaOauthClient fetches an OAuth2 client using the Okta JWK credentials.
func (o *oAuth2Config) fetchOktaOauthClient(ctx context.Context, _ *http.Client) (*http.Client, error) {
	conf := &oauth2.Config{
		ClientID: o.ClientID,
		Scopes:   o.Scopes,
		Endpoint: oauth2.Endpoint{
			TokenURL: o.TokenURL,
		},
	}

	var (
		oktaJWT string
		err     error
	)
	if len(o.OktaJWKPEM) != 0 {
		oktaJWT, err = generateOktaJWTPEM(o.OktaJWKPEM, conf)
		if err != nil {
			return nil, fmt.Errorf("oauth2 client: error generating Okta JWT PEM: %w", err)
		}
	} else {
		oktaJWT, err = generateOktaJWT(o.OktaJWKJSON, conf)
		if err != nil {
			return nil, fmt.Errorf("oauth2 client: error generating Okta JWT: %w", err)
		}
	}

	token, err := exchangeForBearerToken(ctx, oktaJWT, conf)
	if err != nil {
		return nil, fmt.Errorf("oauth2 client: error exchanging Okta JWT for bearer token: %w", err)
	}

	tokenSource := &oktaTokenSource{
		conf:    conf,
		ctx:     ctx,
		oktaJWK: o.OktaJWKJSON,
		token:   token,
	}
	// reuse the tokenSource to refresh the token (automatically calls
	// the custom Token() method when token is no longer valid).
	client := oauth2.NewClient(ctx, oauth2.ReuseTokenSource(token, tokenSource))

	return client, nil
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
	blk, rest := pem.Decode([]byte(pemdata))
	if rest := bytes.TrimSpace(rest); len(rest) != 0 {
		return "", fmt.Errorf("PEM text has trailing data: %s", rest)
	}
	key, err := x509.ParsePKCS8PrivateKey(blk.Bytes)
	if err != nil {
		return "", err
	}
	return signJWT(cnf, key)
}

// signJWT creates a JWT token using required claims and sign it with the
// private key.
func signJWT(cnf *oauth2.Config, key any) (string, error) {
	now := time.Now()
	tok, err := jwt.NewBuilder().Audience([]string{cnf.Endpoint.TokenURL}).
		Issuer(cnf.ClientID).
		Subject(cnf.ClientID).
		IssuedAt(now).
		Expiration(now.Add(time.Hour)).
		Build()
	if err != nil {
		return "", err
	}
	signedToken, err := jwt.Sign(tok, jwt.WithKey(jwa.RS256, key))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return string(signedToken), nil
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
