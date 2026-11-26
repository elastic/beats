// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package jwtauth

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/youmark/pkcs8"
)

var (
	//go:embed testdata/key.pem
	testKeyPEM string

	// testShortSaltKeyPEM is the test key encrypted by openssl with the
	// passphrase "testpass". See TestParsePrivateKey_shortSalt.
	//
	//go:embed testdata/key_encrypted.pem
	testShortSaltKeyPEM string
)

// testPassphrase is long enough for FIPS 140-only mode, which does not accept
// PBKDF2 passwords shorter than 112 bits.
const testPassphrase = "a-test-passphrase" //nolint:gosec // G101: not a credential

// testKeyFingerprint is the fingerprint of the public half of testdata/key.pem
// computed with:
//
//	openssl pkey -in key.pem -pubout |
//	  openssl rsa -pubin -outform DER |
//	  openssl dgst -sha256 -binary | openssl enc -base64 -A
const testKeyFingerprint = "SHA256:/YKB9qFUNXYj4PHZufjgIJJIs3tbMbyKw6mw/jJym6o="

func TestParsePrivateKey(t *testing.T) {
	encrypted := encryptedTestKey(t)
	rsaKey, ok := mustKey(t).(*rsa.PrivateKey)
	if !ok {
		t.Fatalf("test key is %T; want *rsa.PrivateKey", mustKey(t))
	}
	pkcs1 := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(rsaKey)})
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ecDER, err := x509.MarshalECPrivateKey(ecKey)
	if err != nil {
		t.Fatal(err)
	}
	sec1 := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: ecDER})
	legacyEncrypted := pem.EncodeToMemory(&pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: map[string]string{"Proc-Type": "4,ENCRYPTED", "DEK-Info": "AES-128-CBC,0123456789ABCDEF0123456789ABCDEF"},
		Bytes:   []byte("not really encrypted"),
	})
	publicKey := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: []byte("irrelevant")})
	tests := []struct {
		name    string
		pem     string
		pass    string
		wantErr string
	}{
		{name: "plain", pem: testKeyPEM},
		{name: "pkcs1_rsa", pem: string(pkcs1)},
		{name: "sec1_ec", pem: string(sec1)},
		{name: "legacy_encrypted", pem: string(legacyEncrypted), wantErr: "legacy PEM encryption"},
		{name: "wrong_block_type", pem: string(publicKey), wantErr: `unsupported PEM block type "PUBLIC KEY"`},
		{name: "pkcs1_bytes_in_pkcs8_block", pem: strings.Replace(string(pkcs1), "RSA PRIVATE KEY", "PRIVATE KEY", 2), wantErr: "parsing private key:"},
		{name: "encrypted", pem: encrypted, pass: testPassphrase},
		{name: "encrypted_wrong_pass", pem: encrypted, pass: "a-wrong-passphrase", wantErr: "decrypting private key:"},
		// In FIPS 140-only mode this is rejected by the PBKDF2 implementation
		// rather than by the decryption, and must still be an error.
		{name: "encrypted_short_pass", pem: encrypted, pass: "bad", wantErr: "decrypting private key:"},
		{name: "no_pem", pem: "not pem", wantErr: "no PEM data"},
		{name: "trailing_data", pem: testKeyPEM + "extra", wantErr: "trailing data"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ParsePrivateKey([]byte(test.pem), test.pass)
			switch {
			case test.wantErr == "" && err != nil:
				t.Errorf("unexpected error: %v", err)
			case test.wantErr != "" && err == nil:
				t.Errorf("got nil error; want error containing %q", test.wantErr)
			case test.wantErr != "" && !strings.Contains(err.Error(), test.wantErr):
				t.Errorf("error %q does not contain %q", err, test.wantErr)
			}
		})
	}
}

// TestParsePrivateKey_shortSalt checks that testdata/key_encrypted.pem, which
// was made by openssl with its default 64-bit salt and an 8 byte passphrase,
// does not panic. In FIPS 140-only mode the PBKDF2 implementation rejects the
// salt, and the result must be an error. In other modes the key is decrypted.
func TestParsePrivateKey_shortSalt(t *testing.T) {
	_, err := ParsePrivateKey([]byte(testShortSaltKeyPEM), "testpass")
	if err != nil && !strings.Contains(err.Error(), "decrypting private key:") {
		t.Errorf("ParsePrivateKey() error = %v; want nil or an error containing %q", err, "decrypting private key:")
	}
}

func TestPublicKeySHA256(t *testing.T) {
	key := mustKey(t)
	got, err := PublicKeySHA256(key.Public())
	if err != nil {
		t.Fatal(err)
	}
	if got != testKeyFingerprint {
		t.Errorf("PublicKeySHA256() = %q; want %q", got, testKeyFingerprint)
	}
}

func TestSigner(t *testing.T) {
	rsaKey := mustKey(t)
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1700000000, 0)

	tests := []struct {
		name       string
		cfg        SignerConfig
		wantMethod string
		wantClaims map[string]any
		wantKID    string
		wantJTI    bool
		wantErr    string
	}{
		{
			name: "defaults",
			cfg: SignerConfig{
				Key:      rsaKey,
				Lifetime: time.Hour,
				Claims:   map[string]any{"iss": "me", "scope": "a b"},
			},
			wantMethod: "RS256",
			wantClaims: map[string]any{"iss": "me", "scope": "a b", "iat": float64(now.Unix()), "exp": float64(now.Add(time.Hour).Unix())},
		},
		{
			name: "snowflake_placeholder",
			cfg: SignerConfig{
				Key:      rsaKey,
				Lifetime: time.Minute,
				Claims:   map[string]any{"iss": "ACCT.USER.%{public_key_sha256}", "sub": "ACCT.USER"},
			},
			wantMethod: "RS256",
			wantClaims: map[string]any{"iss": "ACCT.USER." + testKeyFingerprint, "sub": "ACCT.USER", "iat": float64(now.Unix()), "exp": float64(now.Add(time.Minute).Unix())},
		},
		{
			name: "kid_jti_rs512",
			cfg: SignerConfig{
				Key:       rsaKey,
				Algorithm: "RS512",
				KeyID:     "k1",
				JTI:       true,
				Lifetime:  time.Minute,
				Claims:    map[string]any{"iss": "me"},
			},
			wantMethod: "RS512",
			wantKID:    "k1",
			wantJTI:    true,
			wantClaims: map[string]any{"iss": "me", "iat": float64(now.Unix()), "exp": float64(now.Add(time.Minute).Unix())},
		},
		{
			name: "ecdsa",
			cfg: SignerConfig{
				Key:       ecKey,
				Algorithm: "ES256",
				Lifetime:  time.Minute,
				Claims:    map[string]any{"iss": "me"},
			},
			wantMethod: "ES256",
			wantClaims: map[string]any{"iss": "me", "iat": float64(now.Unix()), "exp": float64(now.Add(time.Minute).Unix())},
		},
		{name: "no_key", cfg: SignerConfig{Lifetime: time.Minute}, wantErr: "no private key"},
		{name: "no_lifetime", cfg: SignerConfig{Key: rsaKey}, wantErr: "lifetime"},
		{name: "bad_algorithm", cfg: SignerConfig{Key: rsaKey, Algorithm: "HS256", Lifetime: time.Minute}, wantErr: "unsupported algorithm"},
		{name: "key_mismatch", cfg: SignerConfig{Key: rsaKey, Algorithm: "ES256", Lifetime: time.Minute}, wantErr: "signing jwt"},
		{
			name:    "unknown_placeholder",
			cfg:     SignerConfig{Key: rsaKey, Lifetime: time.Minute, Claims: map[string]any{"iss": "%{nope}"}},
			wantErr: `unknown placeholder "%{nope}"`,
		},
		{
			name:    "reserved_claim",
			cfg:     SignerConfig{Key: rsaKey, Lifetime: time.Minute, Claims: map[string]any{"exp": 1}},
			wantErr: `"exp" is set automatically`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s, err := NewSigner(test.cfg)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("NewSigner() error = %v; want error containing %q", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			signed, err := s.Sign(now)
			if err != nil {
				t.Fatal(err)
			}

			claims := jwt.MapClaims{}
			tok, err := jwt.ParseWithClaims(signed, claims, func(*jwt.Token) (any, error) {
				return test.cfg.Key.Public(), nil
			}, jwt.WithTimeFunc(func() time.Time { return now }))
			if err != nil {
				t.Fatalf("verifying jwt: %v", err)
			}
			if got := tok.Method.Alg(); got != test.wantMethod {
				t.Errorf("alg = %q; want %q", got, test.wantMethod)
			}
			if got, _ := tok.Header["kid"].(string); got != test.wantKID {
				t.Errorf("kid = %q; want %q", got, test.wantKID)
			}
			_, hasJTI := claims["jti"]
			if hasJTI != test.wantJTI {
				t.Errorf("jti present = %t; want %t", hasJTI, test.wantJTI)
			}
			delete(claims, "jti")
			if len(claims) != len(test.wantClaims) {
				t.Errorf("claims = %v; want %v", claims, test.wantClaims)
			}
			for k, want := range test.wantClaims {
				if got := claims[k]; got != want {
					t.Errorf("claim %q = %v; want %v", k, got, want)
				}
			}
		})
	}
}

// encryptedTestKey returns the test key encrypted with testPassphrase using a
// 128-bit salt, which FIPS 140-only mode requires.
func encryptedTestKey(t *testing.T) string {
	t.Helper()
	der, err := pkcs8.MarshalPrivateKey(mustKey(t), []byte(testPassphrase), &pkcs8.Opts{
		Cipher: pkcs8.AES256CBC,
		KDFOpts: pkcs8.PBKDF2Opts{
			SaltSize:       16,
			IterationCount: 10000,
			HMACHash:       crypto.SHA256,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "ENCRYPTED PRIVATE KEY", Bytes: der}))
}

func mustKey(t *testing.T) crypto.Signer {
	t.Helper()
	key, err := ParsePrivateKey([]byte(testKeyPEM), "")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := key.(*rsa.PrivateKey); !ok {
		t.Fatalf("test key is %T; want *rsa.PrivateKey", key)
	}
	return key
}
