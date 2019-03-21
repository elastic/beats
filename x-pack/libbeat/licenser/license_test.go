// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package licenser

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLicenseGet(t *testing.T) {
	tests := []struct {
		name string
		t    LicenseType
	}{
		{
			name: "Basic",
			t:    Basic,
		},
		{
			name: "Platinum",
			t:    Platinum,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			l := License{Mode: test.t}
			assert.Equal(t, test.t, l.Get())
		})
	}
}

func TestLicenseIs(t *testing.T) {
	tests := []struct {
		name     string
		t        LicenseType
		query    LicenseType
		expected bool
	}{
		{
			name:     "Basic and asking for Basic",
			t:        Basic,
			query:    Basic,
			expected: true,
		},
		{
			name:     "Platinum and asking for Basic",
			t:        Platinum,
			query:    Basic,
			expected: true,
		},
		{
			name:     "Basic and asking for Platinum",
			t:        Basic,
			query:    Platinum,
			expected: false,
		},
		{
			name:     "Gold and asking for Gold",
			t:        Gold,
			query:    Gold,
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			l := License{Mode: test.t}
			assert.Equal(t, test.expected, l.Cover(test.query))
		})
	}
}

func TestLicenseIsStrict(t *testing.T) {
	tests := []struct {
		name     string
		t        LicenseType
		query    LicenseType
		expected bool
	}{
		{
			name:     "Basic and asking for Basic",
			t:        Basic,
			query:    Basic,
			expected: true,
		},
		{
			name:     "Platinum and asking for Basic",
			t:        Platinum,
			query:    Basic,
			expected: false,
		},
		{
			name:     "Basic and asking for Platinum",
			t:        Basic,
			query:    Platinum,
			expected: false,
		},
		{
			name:     "Gold and asking for Gold",
			t:        Gold,
			query:    Gold,
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			l := License{Mode: test.t}
			assert.Equal(t, test.expected, l.Is(test.query))
		})
	}
}

func TestIsActive(t *testing.T) {
	tests := []struct {
		name     string
		l        License
		expected bool
	}{
		{
			name:     "active",
			l:        License{Status: Active},
			expected: true,
		},
		{
			name:     "inactive",
			l:        License{Status: Inactive},
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.l.IsActive())
		})
	}
}

func TestIsTrial(t *testing.T) {
	tests := []struct {
		name     string
		l        License
		expected bool
	}{
		{
			name:     "is a trial license",
			l:        License{Mode: Trial},
			expected: true,
		},
		{
			name:     "is not a trial license",
			l:        License{Mode: Basic},
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.l.IsTrial())
		})
	}
}

func TestIsTrialExpired(t *testing.T) {
	tests := []struct {
		name     string
		l        License
		expected bool
	}{
		{
			name:     "trial is expired",
			l:        License{Mode: Trial, TrialExpiry: expiryTime(time.Now().Add(-2 * time.Hour))},
			expected: true,
		},
		{
			name:     "trial is not expired",
			l:        License{Mode: Trial, TrialExpiry: expiryTime(time.Now().Add(2 * time.Minute))},
			expected: false,
		},
		{
			name:     "license is not on trial",
			l:        License{Mode: Basic, TrialExpiry: expiryTime(time.Now().Add(2 * time.Minute))},
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.l.IsTrialExpired())
		})
	}
}
