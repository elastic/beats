// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package core

import (
	"errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common/atomic"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/x-pack/libbeat/licenser"
)

var errInvalidLicense = errors.New("invalid license detected, cannot publish events")

// LicenseAwareClient is a client that enforce a specific license, the type implements the
// `License.Watcher` interface and will need to be registered to the license manager.
// The client instance will listen to license change and make sure that the required licenses
// match the current license.
type LicenseAwareClient struct {
	checks []licenser.CheckFunc
	client Client
	log    *logp.Logger
	valid  atomic.Bool
}

// NewLicenseAwareClient returns a new license aware client.
func NewLicenseAwareClient(
	client Client,
	checks ...licenser.CheckFunc,
) *LicenseAwareClient {
	return &LicenseAwareClient{log: logp.NewLogger("license-aware-client"), checks: checks, client: client}
}

// OnNewLicense receives a callback by the license manager when new license is available and control
// if we can send events to the client or not.
func (lac *LicenseAwareClient) OnNewLicense(license licenser.License) {
	valid := licenser.Validate(lac.log, license, lac.checks...)
	lac.valid.Swap(valid)
}

// OnManagerStopped receives a callback from the license manager when the manager is stopped.
func (lac *LicenseAwareClient) OnManagerStopped() {
	// NOOP but need to be implemented for the watcher interface.
}

// PublishAll check if the license allow us to send events.
func (lac *LicenseAwareClient) PublishAll(events []beat.Event) error {
	if lac.valid.Load() {
		return lac.client.PublishAll(events)
	}
	return errInvalidLicense
}

// Publish check if the license allow us to send events.
func (lac *LicenseAwareClient) Publish(event beat.Event) error {
	if lac.valid.Load() {
		return lac.client.Publish(event)
	}
	return errInvalidLicense
}

// Wait proxy the Wait() call to the original client.
func (lac *LicenseAwareClient) Wait() {
	lac.client.Wait()
}

// Close proxy the Close() call to the original client.
func (lac *LicenseAwareClient) Close() error {
	return lac.client.Close()
}
