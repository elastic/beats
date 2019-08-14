// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package core

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/x-pack/libbeat/licenser"
)

type dummySyncClient struct{ EventCount int }

func (dsc *dummySyncClient) Publish(event beat.Event) error {
	dsc.EventCount++
	return nil
}

func (dsc *dummySyncClient) PublishAll(events []beat.Event) error {
	dsc.EventCount += len(events)
	return nil
}

func (dsc *dummySyncClient) Close() error {
	return nil
}

func (dsc *dummySyncClient) Wait() {}

func TestLicenseAwareClient(t *testing.T) {
	t.Run("publish single event", func(t *testing.T) {
		testPublish(t, func(lac *LicenseAwareClient) (int, error) {
			return 1, lac.Publish(beat.Event{})
		})
	})

	t.Run("publish multiple events", func(t *testing.T) {
		testPublish(t, func(lac *LicenseAwareClient) (int, error) {
			return 2, lac.PublishAll([]beat.Event{beat.Event{}, beat.Event{}})
		})
	})
}

func testPublish(t *testing.T, publish func(lac *LicenseAwareClient) (int, error)) {
	// Create strict license check.
	allowBasic := func(log *logp.Logger, l licenser.License) bool {
		return l.Is(licenser.Basic)
	}

	allowPlatinum := func(log *logp.Logger, l licenser.License) bool {
		return l.Is(licenser.Platinum)
	}

	t.Run("when license is valid first check", func(t *testing.T) {
		license := licenser.License{Mode: licenser.Basic}
		client := &dummySyncClient{}
		lac := NewLicenseAwareClient(client, allowBasic, allowPlatinum)
		defer lac.Close()
		lac.OnNewLicense(license)
		count, err := publish(lac)
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, count, client.EventCount)
	})

	t.Run("when license is valid second check", func(t *testing.T) {
		license := licenser.License{Mode: licenser.Platinum}
		client := &dummySyncClient{}
		lac := NewLicenseAwareClient(client, allowBasic, allowPlatinum)
		defer lac.Close()
		lac.OnNewLicense(license)
		count, err := publish(lac)
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, count, client.EventCount)
	})

	t.Run("when license is not valid", func(t *testing.T) {
		license := licenser.License{Mode: licenser.Gold}
		client := &dummySyncClient{}
		lac := NewLicenseAwareClient(client, allowBasic, allowPlatinum)
		defer lac.Close()
		lac.OnNewLicense(license)
		_, err := publish(lac)
		if assert.Error(t, err, errInvalidLicense) {
			return
		}
		assert.Equal(t, 0, client.EventCount)
	})

	t.Run("license is invalid by default", func(t *testing.T) {
		client := &dummySyncClient{}
		lac := NewLicenseAwareClient(client, allowBasic, allowPlatinum)
		defer lac.Close()
		_, err := publish(lac)
		if assert.Error(t, err, errInvalidLicense) {
			return
		}
		assert.Equal(t, 0, client.EventCount)
	})
}
