// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestValidateRequestHMACBodyLimits(t *testing.T) {
	const key = "test-key-longer-than-112-bits"

	sign := func(b []byte) string {
		m := hmac.New(sha256.New, []byte(key))
		m.Write(b)
		return hex.EncodeToString(m.Sum(nil))
	}
	bogus := hex.EncodeToString(make([]byte, sha256.Size))

	tests := []struct {
		name        string
		maxBody     int64
		maxInFlight int64
		bodySize    int
		badSig      bool
		wantStatus  int
		wantErr     error
	}{
		{
			name:       "default_cap_at_limit",
			maxBody:    -1,
			bodySize:   hmacBodyLimit,
			wantStatus: http.StatusAccepted,
		},
		{
			name:       "default_cap_over_limit",
			maxBody:    -1,
			bodySize:   hmacBodyLimit + 1024,
			wantStatus: http.StatusRequestEntityTooLarge,
			wantErr:    errHMACBodyTooLarge,
		},
		{
			// Oversized bodies are rejected as too large whatever their
			// signature is.
			name:       "default_cap_over_limit_bad_signature",
			maxBody:    -1,
			bodySize:   hmacBodyLimit + 1024,
			badSig:     true,
			wantStatus: http.StatusRequestEntityTooLarge,
			wantErr:    errHMACBodyTooLarge,
		},
		{
			name:       "explicit_limit_overrides_default",
			maxBody:    hmacBodyLimit + 10,
			bodySize:   hmacBodyLimit + 1,
			wantStatus: http.StatusAccepted,
		},
		{
			name:       "explicit_limit_truncates",
			maxBody:    10,
			bodySize:   100,
			wantStatus: http.StatusUnauthorized,
			wantErr:    errIncorrectHMACSignature,
		},
		{
			name:        "in_flight_within_limit",
			maxBody:     -1,
			maxInFlight: 1000,
			bodySize:    100,
			wantStatus:  http.StatusAccepted,
		},
		{
			name:        "in_flight_exceeded",
			maxBody:     -1,
			maxInFlight: 50,
			bodySize:    100,
			wantStatus:  http.StatusServiceUnavailable,
			wantErr:     errMaxInFlightExceeded,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var inFlight atomic.Int64
			v := apiValidator{
				method:      http.MethodPost,
				hmacHeader:  "Test-HMAC",
				hmacKey:     key,
				hmacType:    "sha256",
				maxBodySize: test.maxBody,
				inFlight:    &inFlight,
				maxInFlight: test.maxInFlight,
			}

			body := bytes.Repeat([]byte("a"), test.bodySize)
			sig := sign(body)
			if test.badSig {
				sig = bogus
			}
			rd := bytes.NewReader(body)
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", rd)
			req.Header.Set("Test-HMAC", sig)

			gotStatus, err := v.validateRequest(req)
			if gotStatus != test.wantStatus {
				t.Errorf("validateRequest() status = %d; want %d", gotStatus, test.wantStatus)
			}
			if !errors.Is(err, test.wantErr) {
				t.Errorf("validateRequest() error = %v; want %v", err, test.wantErr)
			}
			if test.maxBody < 0 && test.maxInFlight == 0 {
				// The read must stop one byte past the limit, however much
				// the client sends.
				gotRead := len(body) - rd.Len()
				wantRead := min(len(body), hmacBodyLimit+1)
				if gotRead != wantRead {
					t.Errorf("validateRequest() read %d bytes; want %d", gotRead, wantRead)
				}
			}
			if gotStatus == http.StatusAccepted {
				// The body must be left intact for the handler.
				rest, err := io.ReadAll(req.Body)
				if err != nil {
					t.Fatalf("reading restored body: %v", err)
				}
				if !bytes.Equal(rest, body) {
					t.Errorf("restored body has %d bytes; want %d", len(rest), len(body))
				}
			}
			if got := inFlight.Load(); got != 0 {
				t.Errorf("in-flight bytes after validateRequest() = %d; want 0", got)
			}
		})
	}
}
