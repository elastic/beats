// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package dpop

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"math/big"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// ProofGenerator builds DPoP proofs for requests.
// It supports ECDSA, RSA and ED25519 private keys.
type ProofGenerator struct {
	claim   Claimer
	key     crypto.Signer
	jwk     any
	signing jwt.SigningMethod
}

// Claimer wraps the Claims method.
type Claimer interface {
	// Claims returns a jwt.RegisteredClaims that will be added
	// to the DPoP claims held by a ProofClaims constructed by
	// ProofGenerator.BuildProof.
	//
	// When json/v2 is available this will be changed to return jwt.Claims.
	Claims() *jwt.RegisteredClaims
}

// ClaimerFunc is an adapter to allow the use of ordinary functions as a Claimer.
type ClaimerFunc func() *jwt.RegisteredClaims

func (c ClaimerFunc) Claims() *jwt.RegisteredClaims { return c() }

// NewProofGenerator creates a new ProofGenerator.
func NewProofGenerator(claim Claimer, key crypto.Signer, signing jwt.SigningMethod) (*ProofGenerator, error) {
	if claim == nil {
		return nil, errors.New("nil claimer")
	}
	if key == nil {
		return nil, errors.New("nil private key")
	}
	if signing == nil {
		return nil, errors.New("nil signing method")
	}
	jwk, err := buildJWKAndAlg(key)
	if err != nil {
		return nil, err
	}
	return &ProofGenerator{claim: claim, key: key, jwk: jwk, signing: signing}, nil
}

// buildJWKAndAlg constructs a JWK (public key only). Supported keys:
//   - *ecdsa.PrivateKey
//   - *prsa.PrivateKey
//   - *ed25519.PrivateKey
func buildJWKAndAlg(privateKey any) (any, error) {
	switch k := privateKey.(type) {
	case *ecdsa.PrivateKey:
		return ecPublicJWK(&k.PublicKey)
	case *rsa.PrivateKey:
		return rsaPublicJWK(&k.PublicKey)
	case *ed25519.PrivateKey:
		return edPublicJWK(k.Public().(ed25519.PublicKey)) //nolint:errcheck // errcheck is wrong to call this out; if this type assertion ever failed we should probably throw, not just panic.
	default:
		return nil, errors.New("unsupported private key type for DPoP: expected *ecdsa.PrivateKey, *rsa.PrivateKey or *ed25519.PrivateKey")
	}
}

// ecJWK is a minimal ECDSA public key JWK.
type ecJWK struct {
	X   string `json:"x"`
	Y   string `json:"y"`
	Crv string `json:"crv"`
	Kty string `json:"kty"`
}

// ecPublicJWK converts an ECDSA public key into a minimal public JWK.
func ecPublicJWK(pub *ecdsa.PublicKey) (*ecJWK, error) {
	if pub == nil {
		return nil, errors.New("nil ECDSA public key")
	}
	bits := pub.Curve.Params().BitSize
	n := (bits + 7) / 8 // ceil(bits/8)
	return &ecJWK{
		Kty: "EC",
		Crv: pub.Curve.Params().Name,
		X:   base64.RawURLEncoding.EncodeToString(zeroPad(pub.X.Bytes(), n)),
		Y:   base64.RawURLEncoding.EncodeToString(zeroPad(pub.Y.Bytes(), n)),
	}, nil
}

// zeroPad returns a slice of length size, left-padding b with zeros
// if necessary. If len(b) >= size, b is returned unchanged.
func zeroPad(b []byte, size int) []byte {
	if len(b) >= size {
		return b
	}
	p := make([]byte, size)
	copy(p[size-len(b):], b)
	return p
}

// rsaJWK is a minimal RSA public key JWK.
type rsaJWK struct {
	Exponent string `json:"e"`
	Modulus  string `json:"n"`
	Kty      string `json:"kty"`
}

// rsaPublicJWK converts an RSA public key into a minimal public JWK.
func rsaPublicJWK(pub *rsa.PublicKey) (*rsaJWK, error) {
	if pub == nil {
		return nil, errors.New("nil RSA public key")
	}
	return &rsaJWK{
		Kty:      "RSA",
		Modulus:  base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		Exponent: base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
	}, nil
}

// edJWK is a minimal ED25519 public key JWK.
type edJWK struct {
	PublicKey string `json:"x"`
	Kty       string `json:"kty"`
}

// edPublicJWK converts an ED25519 public key into a minimal public JWK.
func edPublicJWK(pub ed25519.PublicKey) (*edJWK, error) {
	if pub == nil {
		return nil, errors.New("nil ED25519 public key")
	}
	return &edJWK{
		Kty:       "OKP",
		PublicKey: base64.RawURLEncoding.EncodeToString(pub),
	}, nil
}

// ProofOptions allows setting optional values such as nonce and access token hash.
type ProofOptions struct {
	// Nonce is set in the DPoP proof as 'nonce' if non-zero.
	Nonce string
	// AccessToken is set as the 'ath' claim after sha256
	// hashing and encoding as raw URL base64.
	AccessToken string
}

// BuildProof returns a signed DPoP proof JWT for the given HTTP method and
// URL. The URL fragment, if present, is stripped per RFC. Optional fields like
// nonce and access token hash (ath) are included when provided via opts.
func (g *ProofGenerator) BuildProof(ctx context.Context, method, url string, opts ProofOptions) (string, error) {
	if g == nil || g.key == nil {
		return "", errors.New("nil proof generator or key")
	}

	htu := url
	if i := strings.Index(htu, "#"); i >= 0 { // strip fragment
		htu = htu[:i]
	}
	claims := ProofClaims{
		RegisteredClaims: g.claim.Claims(),
		Method:           method,
		URL:              htu,
		Nonce:            &opts.Nonce,
	}
	if opts.AccessToken != "" {
		h, err := sha256Base64URL(opts.AccessToken)
		if err != nil {
			return "", err
		}
		claims.AccessTokenHash = &h
	}

	tok := &jwt.Token{
		Header: map[string]any{
			"typ": "dpop+jwt",
			"alg": g.signing.Alg(),
			"jwk": g.jwk,
		},
		Claims: claims,
		Method: g.signing,
	}
	return tok.SignedString(g.key)
}

// ProofClaims represent the standard DPoP proof claims.
type ProofClaims struct {
	// RegisteredClaims is the base set of JWT claims.
	//
	// When json/v2 is available this will be relaxed to jwt.Claims.
	*jwt.RegisteredClaims

	// See https://datatracker.ietf.org/doc/html/rfc9449#section-4.3
	Method          string  `json:"htm"`
	URL             string  `json:"htu"`
	AccessTokenHash *string `json:"ath,omitempty"`
	Nonce           *string `json:"nonce,omitempty"`
}

// sha256Base64URL returns the base64url (no padding) encoding of the SHA-256
// digest of the provided string.
func sha256Base64URL(data string) (string, error) {
	h := sha256.Sum256([]byte(data))
	enc := base64.RawURLEncoding.EncodeToString(h[:])
	return enc, nil
}
