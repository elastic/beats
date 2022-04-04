// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cef

import (
	"time"
)

// Option controls Setting used in unpacking messages.
type Option interface {
	Apply(*Settings)
}

// Settings for unpacking messages.
type Settings struct {
	fullExtensionNames bool

	timezone *time.Location
}

type withFullExtensionNames struct{}

func (w withFullExtensionNames) Apply(s *Settings) {
	s.fullExtensionNames = true
}

// WithFullExtensionNames causes CEF extension key names to be translated into
// their full key names (e.g. src -> sourceAddress).
func WithFullExtensionNames() Option {
	return withFullExtensionNames{}
}

type withTimezone struct {
	timezone *time.Location
}

func (w withTimezone) Apply(s *Settings) {
	s.timezone = w.timezone
}

// WithTimezone causes CEF timestamps that do not contain a timezone to be
// parsed in the specified timezone.
func WithTimezone(timezone *time.Location) Option {
	return withTimezone{timezone}
}
