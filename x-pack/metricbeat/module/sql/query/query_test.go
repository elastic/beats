// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package query

import (
	"errors"
	"fmt"
	"testing"
)

func TestSanitizeError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		sensitive    string
		expectedErr  string
		expectNilErr bool
	}{
		{
			name:         "Nil error",
			err:          nil,
			sensitive:    "password",
			expectedErr:  "",
			expectNilErr: true,
		},
		{
			name:         "Error with sensitive data",
			err:          errors.New("Connection failed: invalid password 'super_secret'"),
			sensitive:    "super_secret",
			expectedErr:  "Connection failed: invalid password '(redacted)'",
			expectNilErr: false,
		},
		{
			name:         "Error with sensitive data (multiple)",
			err:          errors.New("Connection failed: invalid password 'super_secret', cannot parse 'super_secret'"),
			sensitive:    "super_secret",
			expectedErr:  "Connection failed: invalid password '(redacted)', cannot parse '(redacted)'",
			expectNilErr: false,
		},
		{
			name:         "Error with sensitive data (sensitive param contains leading/trailing whitespace)",
			err:          errors.New("Connection failed: invalid password 'super_secret'"),
			sensitive:    "   super_secret ",
			expectedErr:  "Connection failed: invalid password '(redacted)'",
			expectNilErr: false,
		},
		{
			name:         "Sensitive data not found",
			err:          errors.New("No sensitive data present here"),
			sensitive:    "super_secret",
			expectedErr:  "No sensitive data present here",
			expectNilErr: false,
		},
		{
			name:         "Sanitize partial match",
			err:          errors.New("The user admin-admin123 failed authentication"),
			sensitive:    "admin123",
			expectedErr:  "The user admin-(redacted) failed authentication",
			expectNilErr: false,
		},
		{
			name:         "Empty sensitive string",
			err:          errors.New("Nothing should change here"),
			sensitive:    "",
			expectedErr:  "Nothing should change here",
			expectNilErr: false,
		},
		{
			name:         "Sqlserver url parse error",
			err:          fmt.Errorf("cannot open connection: %w", errors.New("testing connection: parse \"sqlserver://mmm\\\\elasticsearch:ttt@localhost:4441\": net/url: invalid userinfo")),
			sensitive:    "sqlserver://mmm\\\\elasticsearch:ttt@localhost:4441",
			expectedErr:  "cannot open connection: testing connection: parse \"(redacted)\": net/url: invalid userinfo",
			expectNilErr: false,
		},
		{
			name:         "Sqlserver url parse error. URL in error is escaped",
			err:          fmt.Errorf("cannot open connection: %w", errors.New("testing connection: parse \"sqlserver://mmm\\\\elasticsearch:ttt@localhost:4441\": net/url: invalid userinfo")),
			sensitive:    "sqlserver://mmm\\elasticsearch:ttt@localhost:4441",
			expectedErr:  "cannot open connection: testing connection: parse (redacted): net/url: invalid userinfo",
			expectNilErr: false,
		},
		{
			name:         "Pattern-based password sanitization in connection string",
			err:          errors.New("Failed to connect: Server=localhost;Database=myDB;User Id=admin;Password=secret123;"),
			sensitive:    "",
			expectedErr:  "Failed to connect: Server=localhost;Database=myDB;User Id=admin;Password=(redacted);",
			expectNilErr: false,
		},
		{
			name:         "Pattern-based URL auth sanitization",
			err:          errors.New("Connection failed for postgres://user:mypassword@localhost:5432/db"),
			sensitive:    "",
			expectedErr:  "Connection failed for postgres://user:(redacted)@localhost:5432/db",
			expectNilErr: false,
		},
		{
			name:         "URL-encoded sensitive data",
			err:          errors.New("Failed to parse: secret%40123"),
			sensitive:    "secret@123",
			expectedErr:  "Failed to parse: (redacted)",
			expectNilErr: false,
		},
		{
			name:         "Multiple password patterns",
			err:          errors.New("pwd=test123 failed, also PASS=another456 failed"),
			sensitive:    "",
			expectedErr:  "pwd=(redacted) failed, also PASS=(redacted) failed",
			expectNilErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := sanitizeError(test.err, test.sensitive)

			if test.expectNilErr && got != nil {
				t.Errorf("sanitizeError() = %v, want nil", got)
				return
			}

			if !test.expectNilErr && got.Error() != test.expectedErr {
				t.Errorf("sanitizeError() = %v, want %v", got.Error(), test.expectedErr)
			}
		})
	}
}
