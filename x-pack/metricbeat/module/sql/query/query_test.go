// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package query

import (
	"errors"
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
