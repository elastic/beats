// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package jwtauth signs JWTs and provides oauth2.TokenSource implementations
// that use them as credentials, either directly or by exchanging them at a
// token endpoint.
package jwtauth

import (
	"bytes"
	"crypto"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/youmark/pkcs8"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/internal/dpop"
)

// ParsePrivateKey decodes a PEM block and parses it as a private key. These
// block types are accepted:
//
//   - "PRIVATE KEY", an unencrypted PKCS#8 key.
//   - "ENCRYPTED PRIVATE KEY", a PKCS#8 key that is decrypted using pass.
//   - "RSA PRIVATE KEY", an unencrypted PKCS#1 RSA key.
//   - "EC PRIVATE KEY", an unencrypted SEC 1 elliptic curve key.
//
// pass is used only for "ENCRYPTED PRIVATE KEY" and may be empty otherwise.
// The older PEM encryption (a Proc-Type header) is not supported.
func ParsePrivateKey(pemdata []byte, pass string) (crypto.Signer, error) {
	blk, rest := pem.Decode(pemdata)
	if blk == nil {
		return nil, errors.New("no PEM data")
	}
	if rest := bytes.TrimSpace(rest); len(rest) != 0 {
		return nil, fmt.Errorf("PEM text has trailing data: %d bytes", len(rest))
	}
	if _, ok := blk.Headers["DEK-Info"]; ok {
		return nil, errors.New("PEM block is encrypted with the legacy PEM encryption, which is not supported; convert it to PKCS#8, for example with openssl pkcs8 -topk8 -v2 aes256")
	}

	var (
		key any
		err error
	)
	switch blk.Type {
	case "ENCRYPTED PRIVATE KEY":
		key, err = decryptPKCS8(blk.Bytes, pass)
		if err != nil {
			return nil, fmt.Errorf("decrypting private key: %w", err)
		}
	case "PRIVATE KEY":
		key, err = x509.ParsePKCS8PrivateKey(blk.Bytes)
	case "RSA PRIVATE KEY":
		key, err = x509.ParsePKCS1PrivateKey(blk.Bytes)
	case "EC PRIVATE KEY":
		key, err = x509.ParseECPrivateKey(blk.Bytes)
	default:
		return nil, fmt.Errorf("unsupported PEM block type %q", blk.Type)
	}
	if err != nil {
		return nil, fmt.Errorf("parsing private key: %w", err)
	}

	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, fmt.Errorf("key is not a signer: %T", key)
	}
	return signer, nil
}

// decryptPKCS8 decrypts and parses an encrypted PKCS#8 private key. In FIPS
// 140-only mode the PBKDF2 implementation panics instead of returning an error
// when the key was encrypted with a salt shorter than 128 bits, which is the
// default for openssl, or when pass is shorter than 112 bits. A panic is turned
// into an error so that a bad key in the configuration does not crash the
// process.
func decryptPKCS8(der []byte, pass string) (key any, err error) {
	defer func() {
		r := recover()
		if r != nil {
			key, err = nil, fmt.Errorf("%v", r)
		}
	}()
	return pkcs8.ParsePKCS8PrivateKey(der, []byte(pass))
}

// PublicKeySHA256 returns the fingerprint of pub in the form used by
// Snowflake key-pair authentication: "SHA256:" followed by the standard
// base64 encoding of the SHA-256 digest of the DER-encoded
// SubjectPublicKeyInfo.
func PublicKeySHA256(pub crypto.PublicKey) (string, error) {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "", fmt.Errorf("marshaling public key: %w", err)
	}
	sum := sha256.Sum256(der)
	return "SHA256:" + base64.StdEncoding.EncodeToString(sum[:]), nil
}

// DefaultAlgorithm is the signing algorithm used when none is configured.
const DefaultAlgorithm = "RS256"

// algorithms is the set of supported signing algorithms.
var algorithms = map[string]jwt.SigningMethod{
	"RS256": jwt.SigningMethodRS256,
	"RS384": jwt.SigningMethodRS384,
	"RS512": jwt.SigningMethodRS512,
	"ES256": jwt.SigningMethodES256,
	"ES384": jwt.SigningMethodES384,
	"ES512": jwt.SigningMethodES512,
}

// reservedClaims are set by the Signer and may not appear in the claims
// template.
var reservedClaims = []string{"iat", "exp", "jti"}

// SignerConfig is the configuration of a Signer.
type SignerConfig struct {
	// Key is the private key used to sign.
	Key crypto.Signer

	// Algorithm is the JWT signing algorithm, for example "RS256".
	// If it is empty, DefaultAlgorithm is used.
	Algorithm string

	// KeyID, if not empty, is added to the JWT header as "kid".
	KeyID string

	// Claims is the template for the JWT claims. String values may contain
	// the placeholder %{public_key_sha256}, which is replaced by the
	// result of PublicKeySHA256 for the public half of Key. The claims
	// iat, exp and jti are set by the Signer and must not be included.
	Claims map[string]any

	// Lifetime is the time between a JWT's iat and exp claims.
	Lifetime time.Duration

	// JTI, if true, adds a random jti claim to each JWT.
	JTI bool
}

// Signer creates signed JWTs. A Signer is safe for concurrent use.
type Signer struct {
	key    crypto.Signer
	method jwt.SigningMethod
	kid    string
	claims map[string]any
	life   time.Duration
	jti    bool
}

// NewSigner returns a Signer for cfg. It fails if the configuration is
// invalid or if the key cannot be used with the algorithm.
func NewSigner(cfg SignerConfig) (*Signer, error) {
	if cfg.Key == nil {
		return nil, errors.New("no private key")
	}
	if cfg.Lifetime <= 0 {
		return nil, errors.New("lifetime must be greater than zero")
	}
	alg := cfg.Algorithm
	if alg == "" {
		alg = DefaultAlgorithm
	}
	method, ok := algorithms[alg]
	if !ok {
		return nil, fmt.Errorf("unsupported algorithm %q", alg)
	}
	for _, k := range reservedClaims {
		if _, ok := cfg.Claims[k]; ok {
			return nil, fmt.Errorf("claim %q is set automatically and must not be configured", k)
		}
	}
	claims, err := expandClaims(cfg.Claims, cfg.Key.Public())
	if err != nil {
		return nil, err
	}

	s := &Signer{
		key:    cfg.Key,
		method: method,
		kid:    cfg.KeyID,
		claims: claims,
		life:   cfg.Lifetime,
		jti:    cfg.JTI,
	}
	// Fail now if the key and algorithm do not match.
	_, err = s.Sign(time.Now())
	if err != nil {
		return nil, err
	}
	return s, nil
}

// Lifetime returns the lifetime of the JWTs created by s.
func (s *Signer) Lifetime() time.Duration { return s.life }

// Sign returns a JWT signed by s with iat set to now and exp set to now plus
// the Signer's lifetime.
func (s *Signer) Sign(now time.Time) (string, error) {
	claims := jwt.MapClaims(maps.Clone(s.claims))
	claims["iat"] = jwt.NewNumericDate(now)
	claims["exp"] = jwt.NewNumericDate(now.Add(s.life))
	if s.jti {
		claims["jti"] = dpop.RandomJTI()
	}
	tok := jwt.NewWithClaims(s.method, claims)
	if s.kid != "" {
		tok.Header["kid"] = s.kid
	}
	signed, err := tok.SignedString(s.key)
	if err != nil {
		return "", fmt.Errorf("signing jwt: %w", err)
	}
	return signed, nil
}

// placeholder matches %{name} in claim values. The syntax avoids ${name},
// which the configuration library interpolates itself.
var placeholder = regexp.MustCompile(`%\{([^}]*)\}`)

// expandClaims returns a copy of claims with placeholders in top-level
// string values replaced.
func expandClaims(claims map[string]any, pub crypto.PublicKey) (map[string]any, error) {
	out := make(map[string]any, len(claims))
	for k, v := range claims {
		s, ok := v.(string)
		if !ok {
			out[k] = v
			continue
		}
		var err error
		out[k], err = expandString(s, pub)
		if err != nil {
			return nil, fmt.Errorf("claim %q: %w", k, err)
		}
	}
	return out, nil
}

func expandString(s string, pub crypto.PublicKey) (string, error) {
	if !strings.Contains(s, "%{") {
		return s, nil
	}
	var err error
	got := placeholder.ReplaceAllStringFunc(s, func(m string) string {
		name := m[2 : len(m)-1]
		if name != "public_key_sha256" {
			err = errors.Join(err, fmt.Errorf("unknown placeholder %q", m))
			return m
		}
		fp, ferr := PublicKeySHA256(pub)
		err = errors.Join(err, ferr)
		return fp
	})
	if err != nil {
		return "", err
	}
	return got, nil
}
