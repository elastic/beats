// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package provider

import (
	"github.com/elastic/beats/v7/libbeat/beat"
)

// SyncDB should ensure the DB is in a state to handle the event before returning.
type Provider interface {
	SyncDB(event *beat.Event, pid uint32) error
}
