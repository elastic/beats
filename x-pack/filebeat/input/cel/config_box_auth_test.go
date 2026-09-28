// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// testBoxPlainKey is a throwaway RSA-2048 key in plain PKCS#8 PEM form,
// generated with x509.MarshalPKCS8PrivateKey for use in tests only.
const testBoxPlainKey = `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCzy2cU1+L8Tpht
Lg9q9nCNqy3pXQQAiuWhGRtBS9pmRo/xl+tlVUeJxN8AVdBjJGjra6pZZ6g8x+NW
tUfuOXNgRa8NUVQIRNkspSQVlb00DD8LQetehnSRQV6I0CY2g4xmobw9246HFIlO
64/KhBF0BxnYx45rJ7CHEuo61p97dzGbvq0yf3aUMKo/NkqxgSLi5dKeOBuolHJb
k/fbRGaXG0YDqpz0EjxnSRZ8PNw35JSNjEpiRhxcLEJo002V+x/SrS/w7peGjdFN
axe47SoKSF2SErtLYxciGEY5Cn3E1A/8DGFXVsIQx987+nzfZJU2Yos/2iHwA/+N
O+u2zGYlAgMBAAECggEACK1pdTUSLHEypBpT/iqUthGr7pZhqhEKEiNfEGCz0rnX
GqblYoeiI0EQLjj2DMLmGW6h0xzQntZa34VySkoVinDyiOcC8j84aBI0UqJedlOc
+1McI/zDRXttL5c0MO9aaF2n8yhUkappEhkGYJTNLtdk5PSEqCFLQMml6l8PZWr/
j8tbbIuNudzpBR8H4/agc0UFZwLK1YpCF9vD9oRZ+HZSum5sM7Hg4UV4UqZUmyaE
Fqdy82P1bw9W2mJ171yYsTP7UzQo6eJ7H0Mdo5AYmReNXLIgE0rPtc1S3KIxWvVl
0k6kp1P0ria11qzjpIuzYZkzE1cLZfuuaVinoznWoQKBgQDRPazPohVCdbc3frT1
a9hBotAAAKg0Mc7eRIb9JqPPj2p5eUanpZgWaQ1To+I/uN+i7EWiCHAAeZhzj8wm
O8H+QEo3jUntIam7TNdEW/TC5Uex4GKshA2EaNekoFudzAcmgrzi3CLsF3e9Ks3a
iOE/1io3NwcAvYb/z+m9tvJrPQKBgQDb+SY+Xwu6wzVPTK6TRPhoBFEDasyzElKt
5i1Jg6K8W0YHe1L9y5jwPVXGPBhN3loRqrO1zJxVwK8CbR13VNagvPw4UEKkkCjh
fuYlNx0ifukAK8fKWqWJm/NyFy5uPqsEh8KrPbkONzKC6oJRxYd3lmZoOYnSr+6u
jwU2BY81CQKBgC7xRkbi1yAs5qjlnVV+F2tKSp3lh9cF4aJN/3bl51RWmY2dHrPX
29ITSXEdUFH5ePrFRS3/9Ji2rvQmK6fcOj5/T+c8pHw11C14JMdqVfQvmjEW5SxN
B/dPyild7I/vSR9jr1q6Bn+vGCbxZnODx/0ZYCk5CDIrUxErJQZx99sFAoGBALGW
co6WExUjNa2gnavdWaI4IeNdXIcROtiT5GneMQpZsa6mnHiy3vTMv6u7pm9vHE34
/v69glUkquWNi+VkA6ZfDEy2VyceDzMFTO4skYPg62Cs963hApWW5rJsDpsIUu7k
X3/546WbYFca1j0H+HbOYDyyfxct28bnRfC4CkZpAoGAeEZ4ZPh2ufpMzKgwWK/U
5lXA5QVH167KYRJR5ptH3GjNjBc6jmsHqb24Dnx9y+7jmVDGSWhRvjhbpj9mT19m
yzbNC9wNkT3IGwoEu1C3SbsstDnPmDILXZXoGODn5jYWYKXwUPRmakkkFpqdkQ4f
A7JN9GvuuK/jTvNqCQDmIb0=
-----END PRIVATE KEY-----
`

// testBoxEncryptedKey is testBoxPlainKey encrypted with passphrase "testpassphrase"
// using pkcs8.MarshalPrivateKey (AES-256-CBC, PBKDF2-SHA256, 16-byte salt).
// The 16-byte salt meets the FIPS 140 minimum of 128 bits.
const testBoxEncryptedKey = `-----BEGIN ENCRYPTED PRIVATE KEY-----
MIIFNTBfBgkqhkiG9w0BBQ0wUjAxBgkqhkiG9w0BBQwwJAQQRClYqexUC+KWMXCl
tRTRFwICCAAwDAYIKoZIhvcNAgkFADAdBglghkgBZQMEASoEEHFjmh5JgPRJ3UAU
L8LAVdUEggTQ0iOxBv/pmJzQIF3s+PJ8eMeU33gZob2ATxEHxkI/VycwUkI9JrFy
LJCW4D9k2r8RVnYT6vlOlVpd8oXHsDuegxOWRmyj7uqfBylT8WHya8Oh7VrhyGQY
TC+NnCh7/IO9kMBXviD0jlCo0OKvWKyQfO4EVWc4oWRklv3gDYto4uUvmrGHbKZP
+W9Zxc9VVc52Rc+AK/zgSNaiDZGBH7Fz2vhkkGGcEAy0Fi/ca+W0ttCfXWvsn8VI
iH9U0Cfzrvs7eejClR5HMwE08AM4w+7lvFxJWGRehXYNiLp1axinaTYXoGq1XhDW
rBad+nlgIKYdzWMhV2qqVUI8NAGp5ez858Bau4YyGsMgO/LONRbhaSVjRnZgdrRz
N3GpdUabkkaFA/Lbkph4aP2wmNcbIrKTzOImsKKZeZn/846CG/AwA/SouITPQOCG
pp8NKtjIth2nkD28gGRPLDHlFgF/QFnRWPq6DNVDVdcAnCSpzyUNf4MYDlR2+6no
JyLC5BTbfV/HY2itZsM5DMIi1br4tlV4ILnjQIgtdFcG0PxhU+kHwVkUPFCo1Rrv
Nq26aB7IVUomTkORluDYhPNZNFkGgm1Hdivce04Za0NJURKgFILh5CgnIvBf5OXe
h75Fpz7qBV6QiO0bi6m0DQTQI3d4GyWRu+zyEccra5ppmhJ6kWRnnzednO397b3n
oR998vgORdzqj/WKdbT0djxKjbQJ+mN+le/U2c0m3qMQQ3pdpA3rO21CSPHiXQIl
T2kri7SXgI0vscUMUyMfN/hs5pu9b1nmor6I9Eil4cR8JGIF2rCJY3PNX7JnvWA2
IXBY6+X1E8ZB+BKdFgsl2swzch1ISSOLdkIo4dUrcpz2R28ui0LoOiOOcBzlF3Me
fir9LXrLj2EoUb6lpUeksMD6vujX7xeCUQoGZ6t0PXtBrSZeUVvRwnAS2ydZeIOY
ddYrMbaLiD8FCZPICweHEq9Dyt7yeWDu4cJiZzLI+R9yXDUkARweUqdK0ioGyC6j
xQzit7ZHRXAlRGnfH+XmDB3grT++08isWY0hjQvD/9gDfjBospePD9DJeI86HzEZ
0vdaFa/3lv/gW288O3SS5lWKcArgsjDiHr5ISpXi5cx5/LW1mTX+AfJD/Pf8Mouv
hcAitZU0e41iTxwAa2CyPgVfsy0eo+MUMFesLK+oJRpeGjVOl7MQRxxfjrRL15Bl
5zlYDypNKL32zDWRBx6NhsZhsNUyrrZ2N1msTQA88R68c4h6gbSljTa/PPoVfecm
gu/o9z/LzYiD8Lrx1YaPUYCmikNRQ0cKMNyYwVZLRJPGd/OJkX/Nyz5vEOfzsXDq
Fsq5JnN5494lOrXcDXRsnz8ojUBg2xNOtZnFEoqzgl8wi6C9o8brWMr0nVaufpUK
DYvs2kY9GMCGk7vq0DSDsBshVN84Nu6UZGhwjro1Fkfeqj4u6BvHygCJp0soNkjZ
uDV2T9eoDYzLPqxlowzMVa2taC2zUgtGKUTbsrLQd+tAgcSnptRxgw9UqiShL7Wh
7HxwgNbm4oqSJf7DLcbmS6tkNBjCburBvKNRKeRdbCaUqgaufdCl6GQ8F3G0H85z
QzeSeBLfa1x+ErLI3fmTQ/hwfENyH+USUQlQKg/YW5piA87ANBW9z5g=
-----END ENCRYPTED PRIVATE KEY-----
`

func TestResolveBoxCredentials(t *testing.T) {
	secret := "a_secret"
	tests := []struct {
		name    string
		cfg     oAuth2Config
		wantErr string
		// fields of the resolved credentials to check
		wantClientID    string
		wantSubjectID   string
		wantSubjectType string
		wantKeyNonNil   bool
	}{
		{
			name: "config_json_enterprise_default",
			cfg: oAuth2Config{
				BoxConfigJSON: []byte(testBoxConfigJSON(t)),
			},
			wantClientID:    "cfg_client_id",
			wantSubjectType: "enterprise",
			wantSubjectID:   "cfg_enterprise_id",
			wantKeyNonNil:   true,
		},
		{
			name: "config_json_user_impersonation",
			cfg: oAuth2Config{
				BoxConfigJSON:  []byte(testBoxConfigJSON(t)),
				BoxSubjectType: "user",
				BoxSubjectID:   "12345",
			},
			wantClientID:    "cfg_client_id",
			wantSubjectType: "user",
			wantSubjectID:   "12345",
			wantKeyNonNil:   true,
		},
		{
			name: "discrete_plain_key",
			cfg: oAuth2Config{
				ClientID:        "disc_id",
				ClientSecret:    &secret,
				BoxPrivateKey:   testBoxPlainKey,
				BoxEnterpriseID: "disc_eid",
			},
			wantClientID:    "disc_id",
			wantSubjectType: "enterprise",
			wantSubjectID:   "disc_eid",
			wantKeyNonNil:   true,
		},
		{
			name: "discrete_encrypted_key",
			cfg: oAuth2Config{
				ClientID:      "disc_id",
				ClientSecret:  &secret,
				BoxPrivateKey: testBoxEncryptedKey,
				BoxPassphrase: "testpassphrase",
			},
			wantClientID:    "disc_id",
			wantSubjectType: "enterprise",
			wantKeyNonNil:   true,
		},
		{
			name: "discrete_encrypted_key_wrong_passphrase",
			cfg: oAuth2Config{
				ClientID:      "disc_id",
				ClientSecret:  &secret,
				BoxPrivateKey: testBoxEncryptedKey,
				BoxPassphrase: "wrong",
			},
			wantErr: "box.private_key: decrypting private key:",
		},
		{
			name: "config_json_bad_key",
			cfg: oAuth2Config{
				BoxConfigJSON: []byte(`{
					"boxAppSettings": {
						"clientID": "x", "clientSecret": "y",
						"appAuth": {"publicKeyID": "k", "privateKey": "not-pem", "passphrase": ""}
					}, "enterpriseID": "e"
				}`),
			},
			wantErr: "box.config_json private key: no PEM data",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Set a dummy token URL so resolveBoxCredentials can proceed.
			test.cfg.TokenURL = boxTokenURL

			got, err := test.cfg.resolveBoxCredentials()
			if test.wantErr != "" {
				if err == nil {
					t.Fatalf("got nil error; want error containing %q", test.wantErr)
				}
				if !strings.Contains(err.Error(), test.wantErr) {
					t.Errorf("error %q does not contain %q", err.Error(), test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.clientID != test.wantClientID {
				t.Errorf("clientID: got %q want %q", got.clientID, test.wantClientID)
			}
			if test.wantSubjectType != "" && got.subjectType != test.wantSubjectType {
				t.Errorf("subjectType: got %q want %q", got.subjectType, test.wantSubjectType)
			}
			if test.wantSubjectID != "" && got.subjectID != test.wantSubjectID {
				t.Errorf("subjectID: got %q want %q", got.subjectID, test.wantSubjectID)
			}
			if test.wantKeyNonNil && got.key == nil {
				t.Error("key is nil; want non-nil")
			}
		})
	}
}

// testBoxConfigJSON returns a synthetic Developer Console config.json using the
// plain test key.
func testBoxConfigJSON(t *testing.T) string {
	t.Helper()
	// JSON-encode the key so newlines are escaped properly.
	b, err := json.Marshal(strings.TrimSpace(testBoxPlainKey))
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	return fmt.Sprintf(`{
  "boxAppSettings": {
    "clientID": "cfg_client_id",
    "clientSecret": "cfg_client_secret",
    "appAuth": {
      "publicKeyID": "cfg_key_id",
      "privateKey": %s,
      "passphrase": ""
    }
  },
  "enterpriseID": "cfg_enterprise_id"
}`, b)
}

func TestSignBoxAssertion(t *testing.T) {
	secret := "a_secret"
	cfg := oAuth2Config{
		ClientID:        "test_client",
		ClientSecret:    &secret,
		BoxPrivateKey:   testBoxPlainKey,
		BoxPublicKeyID:  "my_key_id",
		BoxEnterpriseID: "ent_123",
		TokenURL:        boxTokenURL,
	}
	creds, err := cfg.resolveBoxCredentials()
	if err != nil {
		t.Fatalf("resolveBoxCredentials: %v", err)
	}

	now := time.Now().Truncate(time.Second)
	text, err := signBoxAssertion(creds, now)
	if err != nil {
		t.Fatalf("signBoxAssertion: %v", err)
	}

	var p jwt.Parser
	tok, _, err := p.ParseUnverified(text, jwt.MapClaims{})
	if err != nil {
		t.Fatalf("ParseUnverified: %v", err)
	}

	issuer, err := tok.Claims.GetIssuer()
	if err != nil {
		t.Fatalf("GetIssuer: %v", err)
	}
	if issuer != "test_client" {
		t.Errorf("iss: got %q want %q", issuer, "test_client")
	}

	sub, err := tok.Claims.GetSubject()
	if err != nil {
		t.Fatalf("GetSubject: %v", err)
	}
	if sub != "ent_123" {
		t.Errorf("sub: got %q want %q", sub, "ent_123")
	}

	aud, err := tok.Claims.GetAudience()
	if err != nil {
		t.Fatalf("GetAudience: %v", err)
	}
	if len(aud) != 1 || aud[0] != boxTokenURL {
		t.Errorf("aud: got %v want [%q]", aud, boxTokenURL)
	}

	mc, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("claims are not MapClaims")
	}
	subType, _ := mc["box_sub_type"].(string)
	if subType != "enterprise" {
		t.Errorf("box_sub_type: got %q want %q", subType, "enterprise")
	}

	jtiRaw, _ := mc["jti"].(string)
	if len(jtiRaw) < 16 {
		t.Errorf("jti length %d; want >= 16", len(jtiRaw))
	}

	kid, _ := tok.Header["kid"].(string)
	if kid != "my_key_id" {
		t.Errorf("kid header: got %q want %q", kid, "my_key_id")
	}

	exp, err := tok.Claims.GetExpirationTime()
	if err != nil {
		t.Fatalf("GetExpirationTime: %v", err)
	}
	iat, err := tok.Claims.GetIssuedAt()
	if err != nil {
		t.Fatalf("GetIssuedAt: %v", err)
	}
	if !exp.After(iat.Time) {
		t.Errorf("exp-iat = %v; want > 0", exp.Sub(iat.Time))
	}
	if exp.After(iat.Add(60 * time.Second)) {
		t.Errorf("exp-iat = %v; want <= 60s", exp.Sub(iat.Time))
	}
}

func TestBoxTokenSource_Token(t *testing.T) {
	var callCount int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		body, _ := io.ReadAll(r.Body)
		params, _ := url.ParseQuery(string(body))

		// Verify the form parameters that Box requires.
		if params.Get("grant_type") != "urn:ietf:params:oauth:grant-type:jwt-bearer" {
			t.Errorf("call %d: grant_type: got %q", callCount, params.Get("grant_type"))
		}
		if params.Get("client_id") == "" {
			t.Errorf("call %d: client_id missing", callCount)
		}
		if params.Get("client_secret") == "" {
			t.Errorf("call %d: client_secret missing", callCount)
		}
		if params.Get("assertion") == "" {
			t.Errorf("call %d: assertion missing", callCount)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "mock_token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer srv.Close()

	secret := "a_secret"
	cfg := oAuth2Config{
		ClientID:        "test_client",
		ClientSecret:    &secret,
		BoxPrivateKey:   testBoxPlainKey,
		BoxEnterpriseID: "ent_123",
		TokenURL:        srv.URL,
	}
	creds, err := cfg.resolveBoxCredentials()
	if err != nil {
		t.Fatalf("resolveBoxCredentials: %v", err)
	}

	ts := &boxTokenSource{
		ctx:   context.Background(),
		creds: creds,
	}

	// First call should fetch a new token.
	tok, err := ts.Token()
	if err != nil {
		t.Fatalf("first Token(): %v", err)
	}
	if tok.AccessToken != "mock_token" {
		t.Errorf("access_token: got %q want %q", tok.AccessToken, "mock_token")
	}
	if !tok.Expiry.After(time.Now()) {
		t.Errorf("token expiry %v is not in the future", tok.Expiry)
	}
	if callCount != 1 {
		t.Errorf("callCount after first Token(): got %d want 1", callCount)
	}

	// Second call with a valid token should reuse it without hitting the server.
	_, err = ts.Token()
	if err != nil {
		t.Fatalf("second Token(): %v", err)
	}
	if callCount != 1 {
		t.Errorf("callCount after cached Token(): got %d want 1 (should reuse)", callCount)
	}
}

func TestExchangeBoxAssertion_ClockSkewRetry(t *testing.T) {
	var callCount int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// First call: simulate a server whose clock is 90 seconds ahead.
			serverTime := time.Now().Add(90 * time.Second)
			w.Header().Set("Date", serverTime.UTC().Format(http.TimeFormat))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error":             "invalid_grant",
				"error_description": "exp claim too far in future",
			})
			return
		}
		// Second call: success.
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "mock_token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer srv.Close()

	secret := "a_secret"
	cfg := oAuth2Config{
		ClientID:        "test_client",
		ClientSecret:    &secret,
		BoxPrivateKey:   testBoxPlainKey,
		BoxEnterpriseID: "ent_123",
		TokenURL:        srv.URL,
	}
	creds, err := cfg.resolveBoxCredentials()
	if err != nil {
		t.Fatalf("resolveBoxCredentials: %v", err)
	}

	tok, err := exchangeBoxAssertion(context.Background(), creds, time.Now())
	if err != nil {
		t.Fatalf("exchangeBoxAssertion: %v", err)
	}
	if tok.AccessToken != "mock_token" {
		t.Errorf("access_token: got %q want %q", tok.AccessToken, "mock_token")
	}
	if callCount != 2 {
		t.Errorf("callCount: got %d want 2 (1 failure + 1 retry)", callCount)
	}
}

func TestPemPKCS8PrivateKey(t *testing.T) {
	tests := []struct {
		name    string
		pem     string
		pass    string
		wantErr string
	}{
		{name: "plain", pem: testBoxPlainKey},
		{name: "encrypted", pem: testBoxEncryptedKey, pass: "testpassphrase"},
		{name: "encrypted_wrong_pass", pem: testBoxEncryptedKey, pass: "bad", wantErr: "decrypting private key:"},
		{name: "no_pem", pem: "not pem", wantErr: "no PEM data"},
		{name: "trailing_data", pem: testBoxPlainKey + "extra", wantErr: "trailing data"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := pemPKCS8PrivateKey([]byte(test.pem), test.pass)
			if test.wantErr != "" {
				if err == nil {
					t.Fatalf("got nil error; want error containing %q", test.wantErr)
				}
				if !strings.Contains(err.Error(), test.wantErr) {
					t.Errorf("error %q does not contain %q", err.Error(), test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
