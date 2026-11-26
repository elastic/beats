// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build jwtlive

// These tests run the auth.jwt client against real Snowflake and DocuSign
// endpoints. They are skipped unless the environment variables below are set.
// They are not part of CI: run them locally with
//
//	go test -tags jwtlive -run 'TestLive' -v ./x-pack/filebeat/input/cel/
//
// Snowflake:
//
//	SNOWFLAKE_ACCOUNT          account identifier as it must appear in the
//	                           claims: upper case, "." replaced with "-", no
//	                           region for an account locator
//	SNOWFLAKE_USER             user name, upper case
//	SNOWFLAKE_PRIVATE_KEY_FILE path of the PEM private key
//	SNOWFLAKE_PASSPHRASE       passphrase, if the key is encrypted
//	SNOWFLAKE_EXPECTED_FP      RSA_PUBLIC_KEY_FP from DESCRIBE USER, with or
//	                           without the SHA256: prefix (optional)
//	SNOWFLAKE_HOST             host name, default <account>.snowflakecomputing.com
//	                           in lower case (optional)
//
// DocuSign:
//
//	DOCUSIGN_INTEGRATION_KEY   integration key (iss claim)
//	DOCUSIGN_USER_ID           GUID of the user to impersonate (sub claim)
//	DOCUSIGN_PRIVATE_KEY_FILE  path of the PEM private key
//	DOCUSIGN_PASSPHRASE        passphrase, if the key is encrypted
//	DOCUSIGN_AUTH_HOST         default account-d.docusign.com (developer);
//	                           use account.docusign.com for production
//	DOCUSIGN_MONITOR_URL       a Monitor API URL to request, such as the
//	                           data set stream endpoint (optional)

package cel

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/internal/jwtauth"
)

func TestLiveSnowflake(t *testing.T) {
	account := os.Getenv("SNOWFLAKE_ACCOUNT")
	user := os.Getenv("SNOWFLAKE_USER")
	keyFile := os.Getenv("SNOWFLAKE_PRIVATE_KEY_FILE")
	if account == "" || user == "" || keyFile == "" {
		t.Skip("SNOWFLAKE_ACCOUNT, SNOWFLAKE_USER and SNOWFLAKE_PRIVATE_KEY_FILE are not all set")
	}
	pass := os.Getenv("SNOWFLAKE_PASSPHRASE")
	key := readLiveKey(t, keyFile)
	host := os.Getenv("SNOWFLAKE_HOST")
	if host == "" {
		host = strings.ToLower(account) + ".snowflakecomputing.com"
	}
	const path = "/api/v2/databases"

	cfg := func(alg string, lifetime time.Duration) *jwtAuthConfig {
		return &jwtAuthConfig{
			Mode:       jwtModeDirect,
			PrivateKey: key,
			Passphrase: pass,
			Algorithm:  alg,
			Lifetime:   lifetime,
			Claims: map[string]any{
				"iss": account + "." + user + ".%{public_key_sha256}",
				"sub": account + "." + user,
			},
		}
	}

	t.Run("fingerprint", func(t *testing.T) {
		want := os.Getenv("SNOWFLAKE_EXPECTED_FP")
		if want == "" {
			t.Skip("SNOWFLAKE_EXPECTED_FP is not set")
		}
		signer, err := jwtauth.ParsePrivateKey([]byte(key), pass)
		if err != nil {
			t.Fatal(err)
		}
		got, err := jwtauth.PublicKeySHA256(signer.Public())
		if err != nil {
			t.Fatal(err)
		}
		if got != "SHA256:"+strings.TrimPrefix(want, "SHA256:") {
			t.Errorf("PublicKeySHA256() = %q; want %q", got, want)
		}
	})

	// RS256 is the documented algorithm and must work. The others are
	// reported but do not fail the test, since the SQL API docs only show
	// RS256.
	for _, alg := range []string{"RS256", "RS384", "RS512"} {
		t.Run("request_"+alg, func(t *testing.T) {
			c, err := cfg(alg, 0).client(context.Background(), &http.Client{Timeout: time.Minute})
			if err != nil {
				t.Fatal(err)
			}
			status, body := liveGet(t, c, "https://"+host+path)
			t.Logf("%s: status %d: %.200s", alg, status, body)
			if status != http.StatusOK && alg == "RS256" {
				t.Errorf("status = %d; want 200: %s", status, body)
			}
		})
	}

	t.Run("refresh", func(t *testing.T) {
		// With a 15s lifetime the token expires 1.5s early, and the oauth2
		// package also treats it as expired 10s early, so it is replaced
		// after about 3.5s.
		rec := &recordAuth{base: http.DefaultTransport}
		c, err := cfg("RS256", 15*time.Second).client(context.Background(), &http.Client{Transport: rec, Timeout: time.Minute})
		if err != nil {
			t.Fatal(err)
		}
		for range 2 {
			status, body := liveGet(t, c, "https://"+host+path)
			if status != http.StatusOK {
				t.Fatalf("status = %d; want 200: %s", status, body)
			}
			time.Sleep(5 * time.Second)
		}
		auth := rec.seen()
		if len(auth) != 2 || auth[0] == auth[1] {
			t.Errorf("want two different JWTs; got %d requests, same=%t", len(auth), len(auth) == 2 && auth[0] == auth[1])
		}
	})
}

func TestLiveDocuSign(t *testing.T) {
	iss := os.Getenv("DOCUSIGN_INTEGRATION_KEY")
	sub := os.Getenv("DOCUSIGN_USER_ID")
	keyFile := os.Getenv("DOCUSIGN_PRIVATE_KEY_FILE")
	if iss == "" || sub == "" || keyFile == "" {
		t.Skip("DOCUSIGN_INTEGRATION_KEY, DOCUSIGN_USER_ID and DOCUSIGN_PRIVATE_KEY_FILE are not all set")
	}
	key := readLiveKey(t, keyFile)
	host := os.Getenv("DOCUSIGN_AUTH_HOST")
	if host == "" {
		host = "account-d.docusign.com"
	}
	// The host is used to build the token URL and is the aud claim, so it
	// must be a bare host name. A path or scheme makes DocuSign redirect to an
	// HTML error page, which shows up as a JSON parse error.
	if strings.ContainsAny(host, "/:") {
		t.Fatalf("DOCUSIGN_AUTH_HOST = %q; want a bare host name such as account-d.docusign.com", host)
	}

	cfg := func(clientID string) *jwtAuthConfig {
		return &jwtAuthConfig{
			Mode:       jwtModeGrant,
			PrivateKey: key,
			Passphrase: os.Getenv("DOCUSIGN_PASSPHRASE"),
			TokenURL:   "https://" + host + "/oauth/token",
			ClientID:   clientID,
			Claims: map[string]any{
				"iss":   iss,
				"sub":   sub,
				"aud":   host,
				"scope": "signature impersonation",
			},
		}
	}

	// The exchange is tested without a client secret, as the DocuSign docs
	// show, and with the integration key as the client ID.
	for _, test := range []struct{ name, clientID string }{
		{"no_client_id", ""},
		{"client_id", iss},
	} {
		t.Run("userinfo_"+test.name, func(t *testing.T) {
			c, err := cfg(test.clientID).client(context.Background(), &http.Client{Timeout: time.Minute})
			if err != nil {
				t.Fatal(err)
			}
			status, body := liveGet(t, c, "https://"+host+"/oauth/userinfo")
			t.Logf("status %d: %.300s", status, body)
			if status != http.StatusOK {
				t.Errorf("status = %d; want 200: %s", status, body)
			}
		})
	}

	t.Run("monitor", func(t *testing.T) {
		u := os.Getenv("DOCUSIGN_MONITOR_URL")
		if u == "" {
			t.Skip("DOCUSIGN_MONITOR_URL is not set")
		}
		c, err := cfg("").client(context.Background(), &http.Client{Timeout: time.Minute})
		if err != nil {
			t.Fatal(err)
		}
		status, body := liveGet(t, c, u)
		t.Logf("status %d: %.300s", status, body)
		if status != http.StatusOK {
			t.Errorf("status = %d; want 200: %s", status, body)
		}
	})
}

func readLiveKey(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path) //nolint:gosec // G304: the path is given by the person running the test.
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func liveGet(t *testing.T, c *http.Client, url string) (status int, body string) {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil) //nolint:gosec // G704: the URL is given by the person running the test.
	if err != nil {
		t.Fatal(err)
	}
	// Snowflake accepts this header and infers the type without it.
	req.Header.Set("X-Snowflake-Authorization-Token-Type", "KEYPAIR_JWT")
	resp, err := c.Do(req) //nolint:gosec // G704: the URL is given by the person running the test.
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, string(b)
}

// recordAuth records the Authorization header of each request.
type recordAuth struct {
	base http.RoundTripper

	mu   sync.Mutex
	auth []string
}

func (r *recordAuth) RoundTrip(req *http.Request) (*http.Response, error) {
	r.mu.Lock()
	r.auth = append(r.auth, req.Header.Get("Authorization"))
	r.mu.Unlock()
	return r.base.RoundTrip(req)
}

func (r *recordAuth) seen() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.auth...)
}
